package functions

import (
	"bytes"
	"context"
	"encoding/json"
	"log"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

func NewDriveClient(creds, cfg string) *drive.Service {
	ctx := context.Background()

	config, err := google.ConfigFromJSON([]byte(cfg), drive.DriveReadonlyScope)
	if err != nil {
		log.Fatal(err)
	}

	token := &oauth2.Token{}
	if err := json.NewDecoder(bytes.NewBuffer([]byte(creds))).Decode(token); err != nil {
		log.Fatal(err)
	}

	srv, err := drive.NewService(
		ctx,
		option.WithHTTPClient(config.Client(ctx, token)),
	)
	if err != nil {
		log.Fatal(err)
	}

	return srv
}
