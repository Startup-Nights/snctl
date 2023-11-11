package cmd

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	width  int
	height int

	partnerCmd = &cobra.Command{
		Use:   "partner",
		Short: "Upload one/multiple partner images",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("partner called")

			// TODO: iteratve over directory

			// rename the files

			// resize the files

			//
		},
	}
)

func getDimensions(filename string) (float64, float64, error) {
	var width, height float64

	res, err := exec.Command("inkscape", "-W").CombinedOutput()
	if err != nil {
		return width, height, errors.Wrap(err, "read svg width")
	}

	width, err = strconv.ParseFloat(string(res), 64)
	if err != nil {
		return width, height, errors.Wrap(err, "parse svg width")
	}

	res, err = exec.Command("inkscape", "-H").CombinedOutput()
	if err != nil {
		return width, height, errors.Wrap(err, "read svg height")
	}

	height, err = strconv.ParseFloat(string(res), 64)
	if err != nil {
		return width, height, errors.Wrap(err, "parse svg height")
	}

	return width, height, nil
}

func convertToPng(filename string) (string, error) {
	w, h, err := getDimensions(filename)
	if err != nil {
		return "", errors.Wrap(err, "check svg dimensions")
	}

	// TODO: scale the svg dimensions to a good size
	target := strings.ReplaceAll(filename, ".svg", ".png")

	if err := exec.Command("inkscape", "-w", "-h", filename, "-o", target).Run(); err != nil {
		return "", errors.Wrap(err, "convert svg to png")
	}

	return target, nil
}

func init() {
	uploadCmd.AddCommand(partnerCmd)
	uploadCmd.PersistentFlags().IntVar(&width, "width", 600, "Width to scale the image to")
	uploadCmd.PersistentFlags().IntVar(&height, "height", 300, "Height to scale the image to")
}
