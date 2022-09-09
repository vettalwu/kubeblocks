/*
Copyright © 2022 The dbctl Authors

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

package dbcluster

import (
	"fmt"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/cli-runtime/pkg/resource"
	cmdutil "k8s.io/kubectl/pkg/cmd/util"
	"k8s.io/kubectl/pkg/describe"

	"github.com/apecloud/kubeblocks/pkg/cloudprovider"
	"github.com/apecloud/kubeblocks/pkg/cmd/playground"
	"github.com/apecloud/kubeblocks/pkg/types"
	"github.com/apecloud/kubeblocks/pkg/utils"
)

func NewDescribeCmd(f cmdutil.Factory, streams genericclioptions.IOStreams) *cobra.Command {
	o := &commandOptions{
		FilenameOptions: &resource.FilenameOptions{},
		DescriberSettings: &describe.DescriberSettings{
			ShowEvents: true,
		},

		IOStreams: streams,
	}

	cmd := &cobra.Command{
		Use:   "describe",
		Short: "Describe database cluster info",
		Run: func(cmd *cobra.Command, args []string) {
			cmdutil.CheckErr(o.setup(f, args))
			cmdutil.CheckErr(o.run(
				func(clusterInfo *types.DBClusterInfo) {
					//nolint
					utils.PrintClusterInfo(clusterInfo)
				}, func() error {
					return nil
				}))
		},
	}

	return cmd
}

func buildClusterInfo(obj *unstructured.Unstructured) *types.DBClusterInfo {
	cp := cloudprovider.Get()
	instance, _ := cp.Instance()
	info := types.DBClusterInfo{
		RootUser:    playground.DefaultRootUser,
		DBPort:      playground.DefaultPort,
		DBCluster:   obj.GetName(),
		DBNamespace: obj.GetNamespace(),
		HostIP:      instance.GetIP(),
	}
	for k, v := range obj.GetLabels() {
		info.Labels = info.Labels + fmt.Sprintf("%s:%s ", k, v)
	}

	status := obj.Object["status"].(map[string]interface{})
	cluster := status["cluster"].(map[string]interface{})
	spec := obj.Object["spec"].(map[string]interface{})

	info.Version = spec["version"].(string)
	info.Instances = spec["instances"].(int64)
	info.ServerId = spec["baseServerId"].(int64)
	info.Secret = spec["secretName"].(string)
	info.Status = cluster["status"].(string)
	info.StartTime = ""
	if info.Status == "ONLINE" {
		info.StartTime = status["createTime"].(string)
	}
	info.OnlineInstances = cluster["onlineInstances"].(int64)
	info.Topology = "Cluster"
	if info.Instances == 1 {
		info.Topology = "Standalone"
	}
	info.Engine = playground.DefaultEngine
	info.Storage = 2
	return &info
}
