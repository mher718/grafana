package services

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/grafana/grafana/pkg/cmd/grafana-cli/logger"
	m "github.com/grafana/grafana/pkg/cmd/grafana-cli/models"
)

var (
	IoHelper       m.IoUtil = IoUtilImp{}
	HttpClient     http.Client
	grafanaVersion string
)

func Init(version string) {
	grafanaVersion = version

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
	}

	HttpClient = http.Client{
		Timeout:   time.Duration(10 * time.Second),
		Transport: tr,
	}
}

func ListAllPlugins(repoUrl string) (m.PluginRepo, error) {
	body, err := createRequest(repoUrl, "repo")

	if err != nil {
		logger.Info("Failed to create request", "error", err)
		return m.PluginRepo{}, fmt.Errorf("Failed to create request. error: %v", err)
	}

	if err != nil {
		return m.PluginRepo{}, err
	}

	var data m.PluginRepo
	err = json.Unmarshal(body, &data)
	if err != nil {
		logger.Info("Failed to unmarshal graphite response error: %v", err)
		return m.PluginRepo{}, err
	}

	return data, nil
}

func ReadPlugin(pluginDir, pluginName string) (m.InstalledPlugin, error) {
	distPluginDataPath := path.Join(pluginDir, pluginName, "dist", "plugin.json")

	var data []byte
	var err error
	data, err = IoHelper.ReadFile(distPluginDataPath)

	if err != nil {
		pluginDataPath := path.Join(pluginDir, pluginName, "plugin.json")
		data, err = IoHelper.ReadFile(pluginDataPath)

		if err != nil {
			return m.InstalledPlugin{}, errors.New("Could not find dist/plugin.json or plugin.json on  " + pluginName + " in " + pluginDir)
		}
	}

	res := m.InstalledPlugin{}
	json.Unmarshal(data, &res)

	if res.Info.Version == "" {
		res.Info.Version = "0.0.0"
	}

	if res.Id == "" {
		return m.InstalledPlugin{}, errors.New("could not find plugin " + pluginName + " in " + pluginDir)
	}

	return res, nil
}

func GetLocalPlugins(pluginDir string) []m.InstalledPlugin {
	result := make([]m.InstalledPlugin, 0)
	files, _ := IoHelper.ReadDir(pluginDir)
	for _, f := range files {
		res, err := ReadPlugin(pluginDir, f.Name())
		if err == nil {
			result = append(result, res)
		}
	}

	return result
}

func RemoveInstalledPlugin(pluginPath, pluginName string) error {
	logger.Infof("Removing plugin: %v\n", pluginName)
	pluginDir := path.Join(pluginPath, pluginName)

	_, err := IoHelper.Stat(pluginDir)
	if err != nil {
		return err
	}

	return IoHelper.RemoveAll(pluginDir)
}

func GetPlugin(pluginId, repoUrl string) (m.Plugin, error) {
	body, err := createRequest(repoUrl, "repo", pluginId)

	if err != nil {
		logger.Info("Failed to create request", "error", err)
		return m.Plugin{}, fmt.Errorf("Failed to create request. error: %v", err)
	}

	if err != nil {
		return m.Plugin{}, err
	}

	var data m.Plugin
	err = json.Unmarshal(body, &data)
	if err != nil {
		logger.Info("Failed to unmarshal graphite response error: %v", err)
		return m.Plugin{}, err
	}

	return data, nil
}

func createRequest(repoUrl string, subPaths ...string) ([]byte, error) {
	u, _ := url.Parse(repoUrl)
	for _, v := range subPaths {
		u.Path = path.Join(u.Path, v)
	}

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)

	logger.Info("grafanaVersion ", grafanaVersion)

	req.Header.Set("grafana-version", grafanaVersion)
	req.Header.Set("User-Agent", "grafana "+grafanaVersion)

	if err != nil {
		return []byte{}, err
	}

	res, err := HttpClient.Do(req)

	body, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()

	return body, err
}
