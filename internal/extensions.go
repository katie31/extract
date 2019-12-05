package internal

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"io/ioutil"
	"os"
	"path/filepath"
	"plugin"
)

var Extensions []Extension

type Extension interface {
	registerCommands(cmd *cobra.Command)
	getAllowedConfigKeys() map[string]*string
}

func LoadExtensions(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return err
	}
	for _, file := range files {
		if filepath.Ext(file.Name()) != ".so" {
			continue
		}
		plug, err := plugin.Open(filepath.Join(path, file.Name()))
		if err != nil {
			return errors.Wrap(err, "can't open plugin")
		}

		symExtension, err := plug.Lookup("Extension")
		if err != nil {
			return errors.Wrap(err, "can't find symbol Extension in plugin")
		}
		var extension Extension
		extension, ok := symExtension.(Extension)
		if !ok {
			return errors.New("unexpected type from module symbol")
		}
		Extensions = append(Extensions, extension)
	}
	return nil
}

func RegisterExtensionCommands(rootCmd *cobra.Command) {
	for _, extension := range Extensions {
		extension.registerCommands(rootCmd)
	}
}
