package cmd

import (
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/startup-nights/cli/pkg/functions"
)

var (
	baseDir      string
	targetDir    string
	targetWidth  int
	targetHeight int
	imageType    string

	uploadCmd = &cobra.Command{
		Use:   "upload",
		Short: "Upload images to digitalocean spaces",
		Run: func(cmd *cobra.Command, args []string) {
			timestamp := time.Now().Unix()
			if err := functions.Rename(timestamp, baseDir, imageType); err != nil {
				cobra.CheckErr(errors.Wrap(err, "rename files"))
			}

			if err := functions.Convert(timestamp, baseDir, targetWidth, targetHeight); err != nil {
				cobra.CheckErr(errors.Wrap(err, "convert files between formats"))
			}

			if err := functions.Resize(timestamp, baseDir, targetWidth, targetHeight); err != nil {
				cobra.CheckErr(errors.Wrap(err, "resize files"))
			}

			if err := functions.CopyFiles(timestamp, baseDir, targetDir); err != nil {
				cobra.CheckErr(errors.Wrap(err, "copy files to target folder"))
			}
		},
	}
)

func init() {
	rootCmd.AddCommand(uploadCmd)
	uploadCmd.PersistentFlags().StringVar(&baseDir, "dir", "./", "Directory from which to read from")
	uploadCmd.PersistentFlags().StringVar(&targetDir, "target", os.TempDir(), "Directory from which to read from")
	uploadCmd.PersistentFlags().StringVar(&imageType, "type", "partner", "Type of image aka partner / speaker")
	uploadCmd.PersistentFlags().IntVar(&targetWidth, "width", 600, "Width to scale the image to")
	uploadCmd.PersistentFlags().IntVar(&targetHeight, "height", 300, "Height to scale the image to")
}
