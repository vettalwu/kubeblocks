/*
Copyright ApeCloud, Inc.

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

package components

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	appsv1alpha1 "github.com/apecloud/kubeblocks/apis/apps/v1alpha1"
	intctrlutil "github.com/apecloud/kubeblocks/internal/controllerutil"
	testapps "github.com/apecloud/kubeblocks/internal/testutil/apps"
	testk8s "github.com/apecloud/kubeblocks/internal/testutil/k8s"
)

var _ = Describe("StatefulSet Controller", func() {

	var (
		randomStr          = testCtx.GetRandomStr()
		clusterName        = "mysql-" + randomStr
		clusterDefName     = "cluster-definition-consensus-" + randomStr
		clusterVersionName = "cluster-version-operations-" + randomStr
		opsRequestName     = "wesql-restart-test-" + randomStr
	)

	const (
		revisionID        = "6fdd48d9cd"
		consensusCompName = "consensus"
		consensusCompType = "consensus"
	)

	cleanAll := func() {
		// must wait until resources deleted and no longer exist before the testcases start,
		// otherwise if later it needs to create some new resource objects with the same name,
		// in race conditions, it will find the existence of old objects, resulting failure to
		// create the new objects.
		By("clean resources")

		// delete cluster(and all dependent sub-resources), clusterversion and clusterdef
		testapps.ClearClusterResources(&testCtx)

		// clear rest resources
		inNS := client.InNamespace(testCtx.DefaultNamespace)
		ml := client.HasLabels{testCtx.TestObjLabelKey}
		// namespaced resources
		testapps.ClearResources(&testCtx, intctrlutil.OpsRequestSignature, inNS, ml)
		testapps.ClearResources(&testCtx, intctrlutil.StatefulSetSignature, inNS, ml)
		testapps.ClearResources(&testCtx, intctrlutil.PodSignature, inNS, ml, client.GracePeriodSeconds(0))
	}

	BeforeEach(cleanAll)

	AfterEach(cleanAll)

	testUpdateStrategy := func(updateStrategy appsv1alpha1.UpdateStrategy, componentName string, index int) {
		Expect(testapps.GetAndChangeObj(&testCtx, client.ObjectKey{Name: clusterDefName},
			func(clusterDef *appsv1alpha1.ClusterDefinition) {
				clusterDef.Spec.ComponentDefs[0].ConsensusSpec.UpdateStrategy = appsv1alpha1.SerialStrategy
			})()).Should(Succeed())

		// mock consensus component is not ready
		objectKey := client.ObjectKey{Name: clusterName + "-" + componentName, Namespace: testCtx.DefaultNamespace}
		Expect(testapps.GetAndChangeObjStatus(&testCtx, objectKey, func(newSts *appsv1.StatefulSet) {
			newSts.Status.UpdateRevision = fmt.Sprintf("%s-%s-%dfdd48d8cd", clusterName, componentName, index)
			newSts.Status.ObservedGeneration = newSts.Generation - 1
		})()).Should(Succeed())
	}

	testUsingEnvTest := func(sts *appsv1.StatefulSet) []*corev1.Pod {
		By("mock statefulset update completed")
		updateRevision := fmt.Sprintf("%s-%s-%s", clusterName, consensusCompName, revisionID)
		Expect(testapps.ChangeObjStatus(&testCtx, sts, func() {
			sts.Status.UpdateRevision = updateRevision
			testk8s.MockStatefulSetReady(sts)
			sts.Status.ObservedGeneration = 2
		})).Should(Succeed())

		By("create pods of statefulset")
		pods := testapps.MockConsensusComponentPods(testCtx, sts, clusterName, consensusCompName)

		By("Mock a pod without role label and it will wait for HandleProbeTimeoutWhenPodsReady")
		leaderPod := pods[0]
		Expect(testapps.ChangeObj(&testCtx, leaderPod, func() {
			delete(leaderPod.Labels, intctrlutil.RoleLabelKey)
		})).Should(Succeed())
		Eventually(testapps.CheckObj(&testCtx, client.ObjectKeyFromObject(leaderPod), func(g Gomega, pod *corev1.Pod) {
			g.Expect(pod.Labels[intctrlutil.RoleLabelKey] == "").Should(BeTrue())
		})).Should(Succeed())

		By("mock restart component to trigger reconcile of StatefulSet controller")
		Expect(testapps.ChangeObj(&testCtx, sts, func() {
			sts.Spec.Template.Annotations = map[string]string{
				intctrlutil.RestartAnnotationKey: time.Now().Format(time.RFC3339),
			}
		})).Should(Succeed())

		Eventually(testapps.CheckObj(&testCtx, client.ObjectKeyFromObject(sts),
			func(g Gomega, fetched *appsv1.StatefulSet) {
				g.Expect(fetched.Status.UpdateRevision).To(Equal(updateRevision))
			})).Should(Succeed())

		By("wait for component podsReady to be true and phase to be 'Rebooting'")
		clusterKey := client.ObjectKey{Name: clusterName, Namespace: testCtx.DefaultNamespace}
		Eventually(testapps.CheckObj(&testCtx, clusterKey, func(g Gomega, cluster *appsv1alpha1.Cluster) {
			compStatus := cluster.Status.Components[consensusCompName]
			g.Expect(compStatus.Phase).Should(Equal(appsv1alpha1.RebootingPhase))
			g.Expect(*compStatus.PodsReady).Should(BeTrue())
		})).Should(Succeed())

		By("add leader role label for leaderPod to mock consensus component to be Running")
		Expect(testapps.ChangeObj(&testCtx, leaderPod, func() {
			leaderPod.Labels[intctrlutil.RoleLabelKey] = "leader"
		})).Should(Succeed())
		return pods
	}

	Context("test controller", func() {
		It("test statefulSet controller", func() {
			By("mock cluster object")
			_, _, cluster := testapps.InitConsensusMysql(testCtx, clusterDefName,
				clusterVersionName, clusterName, consensusCompType, consensusCompName)

			By("mock cluster phase is 'Rebooting' and restart operation is running on cluster")
			Expect(testapps.ChangeObjStatus(&testCtx, cluster, func() {
				cluster.Status.Phase = appsv1alpha1.RebootingPhase
				cluster.Status.ObservedGeneration = 1
			})).Should(Succeed())
			_ = testapps.CreateRestartOpsRequest(testCtx, clusterName, opsRequestName, []string{consensusCompName})
			Expect(testapps.ChangeObj(&testCtx, cluster, func() {
				cluster.Annotations = map[string]string{
					intctrlutil.OpsRequestAnnotationKey: fmt.Sprintf(`[{"name":"%s","clusterPhase":"Rebooting"}]`, opsRequestName),
				}
			})).Should(Succeed())

			// trigger statefulset controller Reconcile
			sts := testapps.MockConsensusComponentStatefulSet(testCtx, clusterName, consensusCompName)

			By("wait for component phase to Rebooting by StatefulSet controller when cluster.status.components is nil")
			Eventually(testapps.GetClusterComponentPhase(testCtx, clusterName, consensusCompName)).Should(Equal(appsv1alpha1.RebootingPhase))

			By("mock the StatefulSet and pods are ready")
			// mock statefulSet available and consensusSet component is running
			pods := testUsingEnvTest(sts)

			By("check the component phase becomes Running")
			Eventually(testapps.GetClusterComponentPhase(testCtx, clusterName, consensusCompName)).Should(Equal(appsv1alpha1.RunningPhase))

			By("mock component of cluster is stopping")
			Expect(testapps.GetAndChangeObjStatus(&testCtx, client.ObjectKeyFromObject(cluster), func(tmpCluster *appsv1alpha1.Cluster) {
				tmpCluster.Status.Phase = appsv1alpha1.StoppingPhase
				tmpCluster.Status.Components[consensusCompName] = appsv1alpha1.ClusterComponentStatus{
					Phase: appsv1alpha1.StoppingPhase,
				}
			})()).Should(Succeed())

			By("mock stop operation and processed successfully")
			Expect(testapps.ChangeObj(&testCtx, cluster, func() {
				cluster.Spec.ComponentSpecs[0].Replicas = 0
			})).Should(Succeed())
			Expect(testapps.ChangeObj(&testCtx, sts, func() {
				replicas := int32(0)
				sts.Spec.Replicas = &replicas
			})).Should(Succeed())
			Expect(testapps.ChangeObjStatus(&testCtx, sts, func() {
				testk8s.MockStatefulSetReady(sts)
			})).Should(Succeed())
			// delete all pods of components
			for _, v := range pods {
				testapps.DeleteObject(&testCtx, client.ObjectKeyFromObject(v), v)
			}

			By("check the component phase becomes Stopped")
			Eventually(testapps.GetClusterComponentPhase(testCtx, clusterName, consensusCompName)).Should(Equal(appsv1alpha1.StoppedPhase))

			By("test updateStrategy with Serial")
			testUpdateStrategy(appsv1alpha1.SerialStrategy, consensusCompName, 1)

			By("test updateStrategy with Parallel")
			testUpdateStrategy(appsv1alpha1.ParallelStrategy, consensusCompName, 2)
		})
	})
})