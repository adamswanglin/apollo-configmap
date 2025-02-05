/*
 Copyright 2025 adamswanglin

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
	"strings"
	"time"

	"adamswanglin.github.com/apollo-configmap/internal"
	"adamswanglin.github.com/apollo-configmap/internal/apollosync"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	apolloadamswanglincomv1 "adamswanglin.github.com/apollo-configmap/api/v1"
)

// ApolloConfigServerReconciler reconciles a ApolloConfigServer object
type ApolloConfigServerReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	ConfigStore *apollosync.ConfigStore
}

// +kubebuilder:rbac:groups=apollo.adamswanglin.com,resources=apolloconfigservers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apollo.adamswanglin.com,resources=apolloconfigservers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apollo.adamswanglin.com,resources=apolloconfigservers/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ApolloConfigServer object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.4/pkg/reconcile
func (r *ApolloConfigServerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	logger.Info("Reconciling ApolloConfigServer")
	apolloConfigServer := new(apolloadamswanglincomv1.ApolloConfigServer)
	if err := r.Get(ctx, req.NamespacedName, apolloConfigServer); err != nil {
		logger.Error(err, "Failed to get ApolloConfigServer")
		return internal.RequeueImmediately()
	}
	// list all related ApolloConfig
	apolloConfigList := apolloadamswanglincomv1.ApolloConfigList{}
	// filter spec.apolloConfigServer
	// page list, but not all
	if err := r.List(ctx, &apolloConfigList, client.MatchingLabels{
		"apolloConfigServer": formatLabel(req.NamespacedName.String()),
	}); err != nil {
		logger.Error(err, "Failed to list apolloConfig")
		return internal.RequeueImmediately()
	}

	limitPerSecond := 10
	ticker := time.NewTicker(time.Second / time.Duration(limitPerSecond))
	defer ticker.Stop()

	for _, apolloConfig := range apolloConfigList.Items {
		<-ticker.C
		// update status
		r.ConfigStore.CreateOrUpdateApolloConfig(&apolloConfig, apolloConfigServer)

	}

	return ctrl.Result{}, nil
}

func formatLabel(str string) string {
	return strings.ReplaceAll(str, "/", "-")
}

// SetupWithManager sets up the apollosync with the Manager.
func (r *ApolloConfigServerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&apolloadamswanglincomv1.ApolloConfigServer{}).
		Named("apolloconfigserver").
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}
