/*
Copyright AxonOps Limited 2024.

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

package controller

import (
	"context"
	"time"

	"github.com/axonops/axonops-developer-operator/apps"
	"github.com/axonops/axonops-developer-operator/utils"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	cassandraaxonopscomv1beta1 "github.com/axonops/axonops-developer-operator/api/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// AxonOpsCassandraReconciler reconciles a AxonOpsCassandra object
type AxonOpsCassandraReconciler struct {
	client.Client
	ReconciliationPeriod time.Duration
	Scheme               *runtime.Scheme
	Ctx                  context.Context
}

//+kubebuilder:rbac:groups=cassandra.axonops.com,resources=axonopscassandras,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=cassandra.axonops.com,resources=axonopscassandras/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=cassandra.axonops.com,resources=axonopscassandras/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the AxonOpsCassandra object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.3/pkg/reconcile
func (r *AxonOpsCassandraReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	var axonopsCassCluster cassandraaxonopscomv1beta1.AxonOpsCassandra

	err := r.Get(ctx, req.NamespacedName, &axonopsCassCluster)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	var thisClusterName = axonopsCassCluster.GetName()
	var thisClusterNamespace = axonopsCassCluster.GetNamespace()

	//! [finalizer]
	axonopsFinalizerName := "cassandra.axonops.com/finalizer"
	if axonopsCassCluster.ObjectMeta.DeletionTimestamp.IsZero() {
		if !utils.ContainsString(axonopsCassCluster.GetFinalizers(), axonopsFinalizerName) {
			axonopsCassCluster.SetFinalizers(append(axonopsCassCluster.GetFinalizers(), axonopsFinalizerName))
			if err := r.Update(context.Background(), &axonopsCassCluster); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		statefulSetList := []string{
			"es-" + thisClusterName,
			"ca-" + thisClusterName,
			"as-" + thisClusterName,
		}
		deploymentsList := []string{
			"ds-" + thisClusterName,
		}

		// The object is being deleted
		if utils.ContainsString(axonopsCassCluster.GetFinalizers(), axonopsFinalizerName) {
			// our finalizer is present, so lets handle any external dependency
			for sts := range statefulSetList {
				if err := r.deleteSts(statefulSetList[sts], thisClusterNamespace); err != nil {
					return ctrl.Result{}, client.IgnoreNotFound(err)
				}
				if err := r.deleteSvc(statefulSetList[sts], thisClusterNamespace); err != nil {
					return ctrl.Result{}, client.IgnoreNotFound(err)
				}
			}

			for d := range deploymentsList {
				if err := r.deleteDeployment(deploymentsList[d], thisClusterNamespace); err != nil {
					return ctrl.Result{}, client.IgnoreNotFound(err)
				}
				if err := r.deleteSvc(deploymentsList[d], thisClusterNamespace); err != nil {
					return ctrl.Result{}, client.IgnoreNotFound(err)
				}
			}

			if err := r.deleteIngress("ds-"+thisClusterName, thisClusterNamespace); err != nil {
				return ctrl.Result{}, client.IgnoreNotFound(err)
			}

			// remove our finalizer from the list and update it.
			axonopsCassCluster.SetFinalizers(utils.RemoveString(axonopsCassCluster.GetFinalizers(), axonopsFinalizerName))
			if err := r.Update(context.Background(), &axonopsCassCluster); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}
	//! [finalizer]

	/* Deleted */
	if axonopsCassCluster.GetName() == "" {
		return ctrl.Result{}, nil
	}

	/*
		STEP 1:
		Create the elastic search STS
	*/

	var elasticStatefulSet *appsv1.StatefulSet
	elasticStatefulSet, err = r.getSts("es-"+thisClusterName, thisClusterNamespace)

	if client.IgnoreNotFound(err) != nil {
		return ctrl.Result{}, err
	}

	if elasticStatefulSet == nil {
		/* Create the elastic search STS */
		elasticStatefulSet, err = apps.GenerateElasticsearchConfig(axonopsCassCluster)
		if err != nil {
			return ctrl.Result{}, err
		}

		err = r.Create(ctx, elasticStatefulSet)
		if err != nil {
			return ctrl.Result{}, err
		}
	}
	var elasticSvc *corev1.Service
	elasticSvc, err = r.getService("es-"+thisClusterName, thisClusterNamespace)
	if client.IgnoreNotFound(err) != nil {
		return ctrl.Result{}, err
	}
	if elasticSvc == nil {
		/* Create the elastic search service */
		elasticSvc, err = apps.GenerateElasticsearchServiceConfig(axonopsCassCluster)
		if err != nil {
			return ctrl.Result{}, err
		}

		err = r.Create(ctx, elasticSvc)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	/*
		STEP 2:
		Create the Dashboard Config
	*/

	var dashDeployment *appsv1.Deployment
	dashDeployment, err = r.getDeployment("ds-"+thisClusterName, thisClusterNamespace)

	if client.IgnoreNotFound(err) != nil {
		return ctrl.Result{}, err
	}

	if dashDeployment == nil {
		/* Create the dash search STS */
		dashDeployment, err = apps.GenerateDashboardConfig(axonopsCassCluster)
		if err != nil {
			return ctrl.Result{}, err
		}

		err = r.Create(ctx, dashDeployment)
		if err != nil {
			return ctrl.Result{}, err
		}
	}
	var dashSvc *corev1.Service
	dashSvc, err = r.getService("ds-"+thisClusterName, thisClusterNamespace)
	if client.IgnoreNotFound(err) != nil {
		return ctrl.Result{}, err
	}
	if dashSvc == nil {
		/* Create the dash search service */
		dashSvc, err = apps.GenerateDashboardServiceConfig(axonopsCassCluster)
		if err != nil {
			return ctrl.Result{}, err
		}

		err = r.Create(ctx, dashSvc)
		if err != nil {
			return ctrl.Result{}, err
		}
	}
	if axonopsCassCluster.Spec.AxonOps.Dashboard.Ingress.Enabled {
		/* Create the dash search Ingress */
		var dashIngressCurrent *networkingv1.Ingress
		var dashIngress *networkingv1.Ingress
		dashIngressCurrent, err = r.getIngress("ds-"+thisClusterName, thisClusterNamespace)
		if client.IgnoreNotFound(err) != nil {
			return ctrl.Result{}, err
		}
		dashIngress, err = apps.GenerateDashboardIngressConfig(axonopsCassCluster)
		if err != nil {
			return ctrl.Result{}, err
		}

		if dashIngressCurrent == nil {
			err = r.Create(ctx, dashIngress)
			if err != nil {
				return ctrl.Result{}, err
			}
		} else {
			/* Update the dash search Ingress */
			err = r.Update(ctx, dashIngress)
			if err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	/*
		STEP 3:
		Create the AxonServer Config
	*/

	var axonServerSts *appsv1.StatefulSet
	axonServerSts, err = r.getSts("as-"+thisClusterName, thisClusterNamespace)

	if client.IgnoreNotFound(err) != nil {
		return ctrl.Result{}, err
	}

	if axonServerSts == nil {
		/* Create the axonServer search STS */
		axonServerSts, err = apps.GenerateServerConfig(axonopsCassCluster)
		if err != nil {
			return ctrl.Result{}, err
		}

		err = r.Create(ctx, axonServerSts)
		if err != nil {
			return ctrl.Result{}, err
		}
	}
	var axonServerSvc *corev1.Service
	axonServerSvc, err = r.getService("as-"+thisClusterName, thisClusterNamespace)
	if client.IgnoreNotFound(err) != nil {
		return ctrl.Result{}, err
	}
	if axonServerSvc == nil {
		/* Create the axonServer search service */
		axonServerSvc, err = apps.GenerateServerServiceConfig(axonopsCassCluster)
		if err != nil {
			return ctrl.Result{}, err
		}

		err = r.Create(ctx, axonServerSvc)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	/*
		STEP 4:
		Create the Cassandra STS
	*/

	var cassandraStatefulSetCurrent *appsv1.StatefulSet
	var cassandraStatefulSet *appsv1.StatefulSet
	cassandraStatefulSetCurrent, err = r.getSts("ca-"+thisClusterName, thisClusterNamespace)

	if client.IgnoreNotFound(err) != nil {
		return ctrl.Result{}, err
	}

	/* Create the cassandra search STS */
	cassandraStatefulSet, err = apps.GenerateCassandraConfig(axonopsCassCluster)
	if err != nil {
		return ctrl.Result{}, err
	}

	if cassandraStatefulSetCurrent == nil {
		err = r.Create(ctx, cassandraStatefulSet)
		if err != nil {
			return ctrl.Result{}, err
		}
	} else {
		/* Update the cassandra search STS */
		err = r.Update(ctx, cassandraStatefulSet)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	var cassandraSvc *corev1.Service
	cassandraSvc, err = r.getService("ca-"+thisClusterName, thisClusterNamespace)
	if client.IgnoreNotFound(err) != nil {
		return ctrl.Result{}, err
	}
	if cassandraSvc == nil {
		/* Create the cassandra search service */
		cassandraSvc, err = apps.GenerateCassandraServiceConfig(axonopsCassCluster)
		if err != nil {
			return ctrl.Result{}, err
		}

		err = r.Create(ctx, cassandraSvc)
		if err != nil {
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *AxonOpsCassandraReconciler) SetupWithManager(mgr ctrl.Manager) error {
	pred := predicate.GenerationChangedPredicate{}

	return ctrl.NewControllerManagedBy(mgr).
		For(&cassandraaxonopscomv1beta1.AxonOpsCassandra{}).
		Owns(&appsv1.StatefulSet{}).WithEventFilter(pred).
		Owns(&appsv1.Deployment{}).WithEventFilter(pred).
		Owns(&corev1.Service{}).WithEventFilter(pred).
		Complete(r)
}

func (r *AxonOpsCassandraReconciler) getSts(name string, namespace string) (*appsv1.StatefulSet, error) {
	var statefulSet appsv1.StatefulSet

	err := r.Get(r.Ctx, client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}, &statefulSet)
	if err != nil {
		return nil, err
	}

	return &statefulSet, nil
}

func (r *AxonOpsCassandraReconciler) getDeployment(name string, namespace string) (*appsv1.Deployment, error) {
	var dep appsv1.Deployment

	err := r.Get(r.Ctx, client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}, &dep)
	if err != nil {
		return nil, err
	}

	return &dep, nil
}

func (r *AxonOpsCassandraReconciler) getService(name string, namespace string) (*corev1.Service, error) {
	var svc corev1.Service

	err := r.Get(r.Ctx, client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}, &svc)
	if err != nil {
		return nil, err
	}

	return &svc, nil
}

func (r *AxonOpsCassandraReconciler) deleteSts(name string, namespace string) error {
	statefulSet := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
	}

	return client.IgnoreNotFound(r.Delete(r.Ctx, statefulSet))
}

func (r *AxonOpsCassandraReconciler) deleteDeployment(name string, namespace string) error {
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
	}

	return client.IgnoreNotFound(r.Delete(r.Ctx, dep))
}

func (r *AxonOpsCassandraReconciler) deleteSvc(name string, namespace string) error {
	dep := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
	}

	return client.IgnoreNotFound(r.Delete(r.Ctx, dep))
}

func (r *AxonOpsCassandraReconciler) getIngress(name string, namespace string) (*networkingv1.Ingress, error) {
	var svc networkingv1.Ingress

	err := r.Get(r.Ctx, client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}, &svc)
	if err != nil {
		return nil, err
	}

	return &svc, nil
}

func (r *AxonOpsCassandraReconciler) deleteIngress(name string, namespace string) error {
	dep := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
	}

	return client.IgnoreNotFound(r.Delete(r.Ctx, dep))
}
