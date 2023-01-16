/*
Copyright ApeCloud Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package configuration

import (
	"fmt"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	dbaasv1alpha1 "github.com/apecloud/kubeblocks/apis/dbaas/v1alpha1"
)

type ConfigType string

const (
	CfgCmType    ConfigType = "configmap"
	CfgTplType   ConfigType = "configConstraint"
	CfgLocalType ConfigType = "local"
	CfgRawType   ConfigType = "raw"
)

type RawConfig struct {
	// formatter
	Type dbaasv1alpha1.ConfigurationFormatter

	RawData string
}

type IniContext struct {
	SectionName string
}

// XMLContext TODO(zt) Support Xml config
type XMLContext struct {
}

type CfgOpOption struct {
	// optional
	VolumeName string
	// optional
	FileName string

	// option
	// for all configuration
	AllSearch bool

	// optional
	IniContext *IniContext
	// optional
	XMLContext *XMLContext
}

type ConfigOperator interface {
	// MergeFrom update parameter by keyvalue
	MergeFrom(params map[string]interface{}, option CfgOpOption) error

	// MergeFromConfig(fileContent []byte, option CfgOpOption) error
	// MergePatch(jsonPatch []byte, option CfgOpOption) error
	// Diff(target *ConfigOperator) (*ConfigDiffInformation, error)

	// Query get parameter
	Query(jsonpath string, option CfgOpOption) ([]byte, error)

	// ToCfgContent to configuration file content
	ToCfgContent() (map[string]string, error)
}

type GetResourceFn func(key client.ObjectKey) (map[string]string, error)

type K8sConfig struct {
	CfgKey     client.ObjectKey
	ResourceFn GetResourceFn

	// configmap data
	Configurations map[string]string
}

type CfgOption struct {
	Type ConfigType
	Log  logr.Logger

	// formatter
	CfgType dbaasv1alpha1.ConfigurationFormatter

	// Path for CfgLocalType test
	Path    string
	RawData []byte

	// K8sKey for k8s resource
	K8sKey *K8sConfig
}

// GenerateTPLUniqLabelKeyWithConfig generate uniq key for configuration template
// reference: docs/img/reconfigure-cr-relationship.drawio.png
func GenerateTPLUniqLabelKeyWithConfig(configKey string) string {
	return GenerateUniqKeyWithConfig(ConfigurationTplLabelPrefixKey, configKey)
}

// GenerateUniqKeyWithConfig is similar to getInstanceCfgCMName, generate uniq label or annotations for configuration template
func GenerateUniqKeyWithConfig(label string, configKey string) string {
	return fmt.Sprintf("%s-%s", label, configKey)
}

// GenerateConstraintsUniqLabelKeyWithConfig generate uniq key for configure template
// reference: docs/img/reconfigure-cr-relationship.drawio.png
func GenerateConstraintsUniqLabelKeyWithConfig(configKey string) string {
	return GenerateUniqKeyWithConfig(ConfigurationConstraintsLabelPrefixKey, configKey)
}

// GetInstanceCMName  {{statefull.Name}}-{{appVersion.Name}}-{{tpl.Name}}-"config"
func GetInstanceCMName(obj client.Object, tpl *dbaasv1alpha1.ConfigTemplate) string {
	return getInstanceCfgCMName(obj.GetName(), tpl.VolumeName)
	// return fmt.Sprintf("%s-%s-config", sts.GetName(), tpl.VolumeName)
}

// getInstanceCfgCMName configmap generation rule for configuration file.
// {{statefulset.Name}}-{{volumeName}}
func getInstanceCfgCMName(objName, tplName string) string {
	return fmt.Sprintf("%s-%s", objName, tplName)
}

// GetComponentCfgName is similar to getInstanceCfgCMName, without statefulSet object.
func GetComponentCfgName(clusterName, componentName, tplName string) string {
	return getInstanceCfgCMName(fmt.Sprintf("%s-%s", clusterName, componentName), tplName)
}