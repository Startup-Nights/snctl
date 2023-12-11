package cmd

import (
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/startup-nights/cli/pkg/functions"
)

var (
	baseDir      string
	targetDir    string
	targetWidth  int
	targetHeight int
	imageType    string
	spacesFolder string

	uploadCmd = &cobra.Command{
		Use:   "upload",
		Short: "Upload images to digitalocean spaces",
		Long: `Example usage to upload team members:
go run main.go upload --dir ~/Downloads --folder 2024/team --type team --height 500 --width 500 --target ~/Downloads/cleaned`,
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

			config := functions.SpacesConfig{
				Bucket: viper.GetString("spaces_bucket"),
				Region: viper.GetString("spaces_region"),
				Secret: viper.GetString("spaces_secret"),
				Key:    viper.GetString("spaces_key"),
			}

			if err := functions.Upload(config, targetDir, spacesFolder); err != nil {
				cobra.CheckErr(errors.Wrap(err, "upload files to spaces"))
			}
		},
	}
)

func init() {
	// example:
	// go run main.go upload --dir ~/Downloads --folder 2024/team --type team --height 500 --width 500 --target ~/Downloads/cleaned
	rootCmd.AddCommand(uploadCmd)
	uploadCmd.PersistentFlags().StringVar(&baseDir, "dir", "./", "Directory from which to read from")
	uploadCmd.PersistentFlags().StringVar(&targetDir, "target", "./cleaned", "Directory from which to read from")
	uploadCmd.PersistentFlags().StringVar(&spacesFolder, "folder", "tmp", "Directory on spaces where the files should be saved")
	uploadCmd.PersistentFlags().StringVar(&imageType, "type", "partner", "Type of image aka partner / speaker / team")
	uploadCmd.PersistentFlags().IntVar(&targetWidth, "width", 600, "Width to scale the image to")
	uploadCmd.PersistentFlags().IntVar(&targetHeight, "height", 300, "Height to scale the image to")
}
