/*
Copyright (C) 2022-2023 ApeCloud Co., Ltd

This file is part of KubeBlocks project

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>.
*/

package apps

import (
	appsv1alpha1 "github.com/apecloud/kubeblocks/apis/apps/v1alpha1"
	"github.com/apecloud/kubeblocks/internal/controller/graph"
	ictrltypes "github.com/apecloud/kubeblocks/internal/controller/types"
)

type initTransformer struct {
	cluster       *appsv1alpha1.Cluster
	originCluster *appsv1alpha1.Cluster
}

var _ graph.Transformer = &initTransformer{}

func (t *initTransformer) Transform(ctx graph.TransformContext, dag *graph.DAG) error {
	// put the cluster object first, it will be root vertex of DAG
	rootVertex := &ictrltypes.LifecycleVertex{Obj: t.cluster, ObjCopy: t.originCluster, Action: ictrltypes.ActionStatusPtr()}
	dag.AddVertex(rootVertex)
	return nil
}
