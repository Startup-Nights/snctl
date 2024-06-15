package cmd

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"html/template"
	"io"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/startup-nights/snctl/pkg/functions"
	"google.golang.org/api/googleapi"
)

var (
	teamCmd = &cobra.Command{
		Use:   "team",
		Short: "Upload team members from csv file",
		Run: func(cmd *cobra.Command, args []string) {
			data, err := os.ReadFile(csvFile)
			if err != nil {
				log.Fatal(err)
			}

			records, err := csv.NewReader(bytes.NewBuffer(data)).ReadAll()
			if err != nil {
				log.Fatal(err)
			}

			t, err := template.New("output").Parse(teamMemberTemplate)
			if err != nil {
				log.Fatal(err)
			}

			type member struct {
				Name     string
				Position string
				Linkedin string
				Image    string
			}

			members := []member{}

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
				log.Fatal(err)
			}

			client := s3.New(session)

			for i, record := range records {
				// skip header + empty lines
				if i < 2 || record[0] == "" || record[4] == "" || record[5] == "" {
					continue
				}

				fmt.Printf("uploading %s %s\n", record[0], record[1])

				id := strings.TrimPrefix(record[5], "https://drive.google.com/file/d/")
				id = strings.TrimSuffix(id, "/view?usp=sharing")
				id = strings.TrimSuffix(id, "/view?usp=drive_link")

				file, err := srv.Files.Get(id).Do(
					googleapi.QueryParameter("supportsAllDrives", "True"),
				)
				if err != nil {
					log.Printf("get %s %s file info: %v", record[0], record[1], err)
					log.Fatal(err)
				}

				res, err := srv.Files.Get(id).Download(
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

				filename := functions.SimplifyName(file.Name)

				data, err = functions.ResizeImage(data, filename, 500, 500)
				if err != nil {
					log.Printf("resize image: %v", err)
					log.Fatal(err)
				}

				url, err := functions.UploadImage(client, cfg.Bucket, filename, "2024/team", data)
				if err != nil {
					log.Printf("upload image: %v", err)
					log.Fatal(err)
				}

				members = append(members, member{
					Name:     fmt.Sprintf("%s %s", record[0], record[1]),
					Position: record[3],
					Linkedin: record[4],
					Image:    url,
				})
			}

			var buf bytes.Buffer

			if err := t.Execute(&buf, struct {
				Members []member
			}{
				Members: members,
			}); err != nil {
				log.Fatal(err)
			}

			fmt.Println(buf.String())
		},
	}
)

func init() {
	uploadCmd.AddCommand(teamCmd)
}

var teamMemberTemplate = `
{{ range .Members }}
- name: '{{ .Name }}'
            position: '{{ .Position }}'
            linkedin: '{{ .Linkedin }}'
            src: '{{ .Image }}'{{ end }}
`
