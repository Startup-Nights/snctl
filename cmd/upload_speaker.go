package cmd

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/startup-nights/snctl/pkg/functions"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/googleapi"
)

var (
	speakerCmd = &cobra.Command{
		Use:   "speaker",
		Short: "Upload speakers from csv file",
		Run: func(cmd *cobra.Command, args []string) {
			data, err := os.ReadFile(csvFile)
			if err != nil {
				log.Printf("readfile: %v", err)
				log.Fatal(err)
			}

			records, err := csv.NewReader(bytes.NewBuffer(data)).ReadAll()
			if err != nil {
				log.Printf("read csv: %v", err)
				log.Fatal(err)
			}

			t, err := template.New("output").Parse(speakerTemplate)
			if err != nil {
				log.Printf("parse template: %v", err)
				log.Fatal(err)
			}

			type speaker struct {
				Name        string
				Position    string
				Image       string
				Description string
			}

			speakers := []speaker{}

			srv := functions.NewDriveClient(
				viper.GetString("drive_token"),
				viper.GetString("credentials"),
			)

			cfg := functions.SpacesConfig{
				Bucket: viper.GetString("spaces_bucket"),
				Region: viper.GetString("spaces_region"),
				Secret: viper.GetString("spaces_secret"),
				Key:    viper.GetString("spaces_key"),
			}

			config := &aws.Config{
				Credentials: credentials.NewStaticCredentials(cfg.Key, cfg.Secret, ""),
				Endpoint:    aws.String(fmt.Sprintf("%s.digitaloceanspaces.com:443", strings.TrimSpace(cfg.Region))),
				Region:      aws.String(cfg.Region),
			}

			session, err := session.NewSession(config)
			if err != nil {
				log.Printf("create new session: %v", err)
				log.Fatal(err)
			}

			client := s3.New(session)

			// get all currently uploaded speaker images
			files, err := srv.Files.List().Q("'11Tqb7iAW8QUpqw2TaSu55LqQ-RhrgEWr' in parents").Do(
				googleapi.QueryParameter("supportsAllDrives", "True"),
				googleapi.QueryParameter("includeItemsFromAllDrives", "True"),
			)
			if err != nil {
				log.Printf("list files: %v", err)
				log.Fatal(err)
			}
			if len(files.Files) == 0 {
				log.Fatal("no images found in drive folder")
				return
			}

			for i, record := range records {
				// skip header
				// skip speakers without name
				// skip speakers who didn't fill out the form yet
				// skip speakers who haven't uploaded an image yet
				if i < 2 || record[0] == "" || record[14] == "" || record[18] == "" {
					continue
				}

				var image *drive.File
				found := false

				name := record[0]
				fmt.Printf("uploading %s\n", name)

				for _, file := range files.Files {
					if file.Name == name {
						found = true
						image = file
					}
				}

				if !found {
					fmt.Printf("did not find matching image for %s\n", name)
					continue
				}

				res, err := srv.Files.Get(image.Id).Download(
					googleapi.QueryParameter("supportsAllDrives", "True"),
				)
				if err != nil {
					log.Printf("download %s %s image: %v", record[0], record[1], err)
					log.Fatal(err)
				}

				data, err := io.ReadAll(res.Body)
				if err != nil {
					log.Printf("read response body: %v", err)
					log.Fatal(err)
				}

				input := filepath.Join(os.TempDir(), "input")
				output := filepath.Join(os.TempDir(), "output.png")

				if err := os.WriteFile(input, data, 0644); err != nil {
					log.Printf("write file: %v", err)
					log.Fatal(err)
				}

				if err := exec.Command("convert", input, output).Run(); err != nil {
					log.Printf("convert: %v", err)
					log.Fatal(err)
				}

				data, err = os.ReadFile(output)
				if err != nil {
					log.Printf("readfile: %v", err)
					log.Fatal(err)
				}

				filename := functions.SimplifyName(image.Name)

				switch filepath.Ext(filename) {
				case "":
					filename += ".png"
				default:
					strings.ReplaceAll(filename, filepath.Ext(filename), ".png")
				}

				data, err = functions.ResizeImage(data, filename, 500, 500)
				if err != nil {
					log.Printf("resize image: %v", err)
					log.Fatal(err)
				}

				url, err := functions.UploadImage(client, cfg.Bucket, filename, "2024/speaker", data)
				if err != nil {
					log.Printf("upload image: %v", err)
					log.Fatal(err)
				}

				speakers = append(speakers, speaker{
					Name:        record[0],
					Position:    record[10],
					Description: record[26],
					Image:       url,
				})
			}

			var buf bytes.Buffer

			if err := t.Execute(&buf, struct {
				Speakers []speaker
			}{
				Speakers: speakers,
			}); err != nil {
				log.Printf("template: %v", err)
				log.Fatal(err)
			}

			fmt.Println(buf.String())
		},
	}
)

func init() {
	uploadCmd.AddCommand(speakerCmd)
}

var speakerTemplate = `
{{ range .Speakers }}
            - name: '{{ .Name }}'
              position: '{{ .Position }}'
              description: >-
                {{ .Description }}
              image:
                src: '{{ .Image }}'
                alt: '{{ .Name }}'{{ end }}
`
