package cmd

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/google/go-github/v56/github"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/crypto/blake2b"
	"golang.org/x/crypto/nacl/box"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
)

var (
	renewGmailToken          bool
	renewSheetsToken         bool
	updateEnvironmentSecrets bool
	config                   *oauth2.Config

	updateCmd = &cobra.Command{
		Use:   "update",
		Short: "Renew the tokens and update them in the config file",
		Run: func(cmd *cobra.Command, args []string) {
			if renewGmailToken || renewSheetsToken {
				if !viper.IsSet("credentials") {
					cobra.CheckErr(errors.New("no sheets token file configured"))
				}

				// set up a waitgroup to make sure that - in case we update both
				// tokens - we don't overwrite the global oauth object. Earlier
				// this was not necessary because it required the input from the
				// user
				wg := &sync.WaitGroup{}

				// setup a server to extract the auth token from the url
				// the url itself can be adjusted in the 'credentials' - there is a
				// field "redirect" or something like that
				r := chi.NewRouter()
				r.Get("/", func(w http.ResponseWriter, r *http.Request) {
					defer wg.Done()

					scope := r.URL.Query().Get("scope")
					code := r.URL.Query().Get("code")

					token, err := config.Exchange(context.TODO(), code)
					if err != nil {
						_, _ = w.Write([]byte("failed to get token from code: " + err.Error()))
					}

					var buf bytes.Buffer
					if err := json.NewEncoder(&buf).Encode(token); err != nil {
						_, _ = w.Write([]byte("failed to encode token to json: " + err.Error()))
					}

					switch scope {
					case "https://www.googleapis.com/auth/gmail.compose":
						viper.Set("gmail_token", buf.String())
						_, _ = w.Write([]byte("updated gmail token"))

					case "https://www.googleapis.com/auth/spreadsheets":
						viper.Set("sheets_token", buf.String())
						_, _ = w.Write([]byte("updated sheets token"))

					default:
						_, _ = w.Write([]byte("unknown scope: " + scope))
					}
				})

				go func() {
					_ = http.ListenAndServe(":3333", r)
				}()

				b := []byte(viper.GetString("credentials"))

				if renewSheetsToken {
					var err error
					wg.Add(1)
					config, err = google.ConfigFromJSON(b, "https://www.googleapis.com/auth/spreadsheets")
					if err != nil {
						cobra.CheckErr(errors.Wrap(err, "parse sheets client secret"))
					}
					fmt.Println("=> open this link in your browser: " + config.AuthCodeURL("state-token", oauth2.AccessTypeOffline))
					wg.Wait()
					log.Println("=> generated new sheets token")

					fmt.Println("=> token can be updated here: " + viper.GetString("secrets_url"))
				}

				if renewGmailToken {
					var err error
					wg.Add(1)
					config, err = google.ConfigFromJSON(b, gmail.GmailComposeScope)
					if err != nil {
						cobra.CheckErr(errors.Wrap(err, "parse gmail client secret"))
					}
					fmt.Println("=> open this link in your browser: " + config.AuthCodeURL("state-token", oauth2.AccessTypeOffline))
					wg.Wait()
					log.Println("=> generated new gmail token")
				}

				if err := viper.WriteConfig(); err != nil {
					cobra.CheckErr(errors.Wrap(err, "update config with new tokens"))
				}
			}

			if updateEnvironmentSecrets {
				if !viper.IsSet("github_token") {
					cobra.CheckErr(errors.New("no github token configured"))
				}

				// the repo id can be found in the page source of the repository:
				// <meta content="your-repo-id" name="octolytics-dimension-repository_id" />
				ctx := context.TODO()
				client := github.NewClient(nil).WithAuthToken(viper.GetString("github_token"))

				pub, _, err := client.Actions.GetEnvPublicKey(ctx, 646451604, "prod")
				if err != nil {
					cobra.CheckErr(errors.Wrap(err, "get environment public key"))
				}

				if !viper.IsSet("gmail_token") || !viper.IsSet("sheets_token") {
					cobra.CheckErr(errors.New("gmail oder sheets token is missing in config"))
				}

				secret, err := encrypt(viper.GetString("gmail_token"), pub.GetKey())
				if err != nil {
					cobra.CheckErr(errors.Wrap(err, "encrypt gmail secret"))
				}
				_, err = client.Actions.CreateOrUpdateEnvSecret(ctx, 646451604, "prod", &github.EncryptedSecret{
					Name:           "GMAIL",
					KeyID:          *pub.KeyID,
					EncryptedValue: secret,
				})
				if err != nil {
					cobra.CheckErr(errors.Wrap(err, "update gmail token in 'prod' env"))
				}

				secret, err = encrypt(viper.GetString("sheets_token"), pub.GetKey())
				if err != nil {
					cobra.CheckErr(errors.Wrap(err, "encrypt sheets secret"))
				}
				_, err = client.Actions.CreateOrUpdateEnvSecret(ctx, 646451604, "prod", &github.EncryptedSecret{
					Name:           "SHEETS",
					KeyID:          *pub.KeyID,
					EncryptedValue: secret,
				})
				if err != nil {
					cobra.CheckErr(errors.Wrap(err, "update sheets token in 'prod' env"))
				}

				_, err = client.Actions.CreateWorkflowDispatchEventByFileName(ctx,
					"startup-nights",
					"functions",
					"deploy.yml",
					github.CreateWorkflowDispatchEventRequest{
						Ref: "main",
					},
				)
				if err != nil {
					cobra.CheckErr(errors.Wrap(err, "trigger workflow run"))
				}
			}
		},
	}
)

func encrypt(secret, pubkey string) (string, error) {
	// https://jefflinse.io/posts/encrypting-github-secrets-using-go/
	b, err := base64.StdEncoding.DecodeString(pubkey)
	if err != nil {
		return "", errors.Wrap(err, "decode public key to base64")
	}
	recipientKey := new([32]byte)
	copy(recipientKey[:], b)
	pubKey, privKey, err := box.GenerateKey(rand.Reader)
	if err != nil {
		return "", errors.Wrap(err, "generate key from random data")
	}

	nonceHash, err := blake2b.New(24, nil)
	if err != nil {
		return "", errors.Wrap(err, "create nonce hash")
	}

	nonceHash.Write(pubKey[:])
	nonceHash.Write(recipientKey[:])

	nonce := new([24]byte)
	copy(nonce[:], nonceHash.Sum(nil))

	out := box.Seal(pubKey[:], []byte(secret), nonce, recipientKey, privKey)
	return base64.StdEncoding.EncodeToString(out), nil
}

func init() {
	tokenCmd.AddCommand(updateCmd)
	updateCmd.PersistentFlags().BoolVar(&renewGmailToken, "gmail", false, "A help for foo")
	updateCmd.PersistentFlags().BoolVar(&renewSheetsToken, "sheets", false, "A help for foo")
	updateCmd.PersistentFlags().BoolVar(&updateEnvironmentSecrets, "update-secrets", false, "Update the github actions environment secrets")
}
