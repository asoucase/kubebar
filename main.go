package main

import (
	_ "embed"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/getlantern/systray"
	"gopkg.in/yaml.v2"
)

//go:embed icon.png
var iconData []byte

var (
	configData      map[string]interface{}
	currentCtxLabel = "Active context: %s"
)

func init() {
	data, err := readKubeConfig()
	if err != nil {
		log.Fatalln(err)
	}
	configData = *data
}

func main() {
	systray.Run(onReady, func() {})
}

func onReady() {
	currentCtx, err := getCurrentContext()
	if err != nil {
		log.Fatalln(err)
	}

	contexts, err := getAllContexts()
	if err != nil {
		log.Fatalln(err)
	}

	systray.SetTemplateIcon(iconData, iconData)
	systray.SetTitle("k8s")
	systray.SetTooltip("Kubernetes context switch")

	cLabel := fmt.Sprintf(currentCtxLabel, currentCtx)
	mContext := systray.AddMenuItem(cLabel, cLabel)
	systray.AddSeparator()

	for _, ctx := range contexts {
		m := systray.AddMenuItem(ctx, ctx)

		go func(m *systray.MenuItem, currentCtx string) {
			for {
				<-m.ClickedCh
				mContext.SetTitle(fmt.Sprintf(currentCtxLabel, currentCtx))
				err := switchContext(currentCtx)
				if err != nil {
					log.Fatalln(err)
				}
			}
		}(m, ctx)
	}
	systray.AddSeparator()

	mQuit := systray.AddMenuItem("Quit", "Quit app")

	for {
		<-mQuit.ClickedCh
		systray.Quit()
	}
}

func kubeConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/.kube/config", homeDir), nil
}

func readKubeConfig() (*map[string]interface{}, error) {
	path, err := kubeConfigPath()
	if err != nil {
		return nil, err
	}

	yFile, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	data := make(map[string]interface{})
	err = yaml.Unmarshal(yFile, &data)
	if err != nil {
		return nil, err
	}

	return &data, nil
}

func getCurrentContext() (string, error) {
	currentCtx, ok := configData["current-context"]
	if !ok {
		return "", errors.New("unable to read current-context from kubeconfig")
	}
	return currentCtx.(string), nil
}

func getAllContexts() ([]string, error) {
	ctxs, ok := configData["contexts"]
	if !ok {
		return nil, errors.New("malformed kubeconfig")
	}

	contexts := make([]string, 0)

	for _, ctx := range ctxs.([]interface{}) {
		mCtx := ctx.(map[interface{}]interface{})
		if name, ok := mCtx[interface{}("name")]; ok {
			contexts = append(contexts, name.(string))
		}
	}

	return contexts, nil
}

func switchContext(context string) error {
	data, err := readKubeConfig()
	if err != nil {
		return err
	}

	(*data)["current-context"] = interface{}(context)

	out, err := yaml.Marshal(data)
	if err != nil {
		return err
	}

	path, err := kubeConfigPath()
	if err != nil {
		return err
	}

	return os.WriteFile(path, out, 0o600)
}
