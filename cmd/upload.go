package cmd

import (
	"github.com/spf13/cobra"
)

var (
	resizeImages bool
	renameImages bool
	baseDir      string

	uploadCmd = &cobra.Command{
		Use:   "upload",
		Short: "Upload images to digitalocean spaces",
	}
)

func init() {
	rootCmd.AddCommand(uploadCmd)
	uploadCmd.PersistentFlags().BoolVar(&resizeImages, "resize", true, "Resize the image before uploading it")
	uploadCmd.PersistentFlags().BoolVar(&renameImages, "rename", true, "Rename the images before uploading them")
	uploadCmd.PersistentFlags().StringVar(&baseDir, "dir", "./", "Directory from which to read from")
}
