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
	"adamswanglin.github.com/apollo-configmap/internal"
	"adamswanglin.github.com/apollo-configmap/internal/apollosync"
	"context"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	log2 "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	apollov1 "adamswanglin.github.com/apollo-configmap/api/v1"
)

const SYNC_SYNCING = "Syncing"
const SYNC_SUCCESS = "Success"

// ApolloConfigReconciler reconciles a ApolloConfig object
type ApolloConfigReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	ConfigStore *apollosync.ConfigStore
}

type apolloConfigContext struct {
	context.Context
	apolloConfig       *apollov1.ApolloConfig
	apolloConfigServer *apollov1.ApolloConfigServer
}

// +kubebuilder:rbac:groups=apollo.adamswanglin.com,resources=apolloconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apollo.adamswanglin.com,resources=apolloconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apollo.adamswanglin.com,resources=apolloconfigs/finalizers,verbs=update
// +kubebuilder:rbac:groups=apollo.adamswanglin.com,resources=apolloconfigservers,verbs=get;list;watch;
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// the ApolloConfig object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.4/pkg/reconcile
func (r *ApolloConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log2.FromContext(ctx)
	aCtx := apolloConfigContext{
		Context:            ctx,
		apolloConfig:       new(apollov1.ApolloConfig),
		apolloConfigServer: new(apollov1.ApolloConfigServer),
	}

	log.Info("Reconciling ApolloConfig")
	if err := r.Get(aCtx, req.NamespacedName, aCtx.apolloConfig); err != nil {
		if apierrs.IsNotFound(err) {
			log.Info("ApolloConfig not found. Ignoring since object must be deleted")
			return internal.NoRequeue()
		} else {
			log.Info("Unable to get ApolloConfig", "error", err.Error())
			return internal.RequeueImmediately()
		}
	}
	if internal.HasDeletionTimestamp(aCtx.apolloConfig.ObjectMeta) {
		r.ConfigStore.DeleteApolloConfig(*aCtx.apolloConfig)
		return internal.NoRequeue()
	}

	apolloConfigServer, err := internal.KeyToNamespacedName(aCtx.apolloConfig.Spec.ApolloConfigServer)
	if err != nil {
		err = errors.New("Unable to parse ApolloConfigServer key " + aCtx.apolloConfig.Spec.ApolloConfigServer + "; key must bu full namespacedName eg. namespace/name")
		_ = r.updateStatusAndReturnError(aCtx, SYNC_SYNCING, "", err)
		return internal.RequeueImmediately()
	}
	//update label, used for filter
	if aCtx.apolloConfig.Labels == nil {
		aCtx.apolloConfig.Labels = map[string]string{
			"apolloConfigServer": "",
		}
	}
	if label, exist := aCtx.apolloConfig.Labels["apolloConfigServer"]; !exist || label != aCtx.apolloConfig.Spec.ApolloConfigServer {
		aCtx.apolloConfig.Labels["apolloConfigServer"] = formatLabel(aCtx.apolloConfig.Spec.ApolloConfigServer)
		//update label
		if err := r.Update(aCtx, aCtx.apolloConfig); err != nil {
			_ = r.updateStatusAndReturnError(aCtx, SYNC_SYNCING, "", err)
			return internal.RequeueImmediately()
		}
	}

	if err := r.Get(aCtx, *apolloConfigServer, aCtx.apolloConfigServer); err != nil {
		_ = r.updateStatusAndReturnError(aCtx, SYNC_SYNCING, "", err)
		return internal.RequeueImmediately()
	}
	r.ConfigStore.CreateOrUpdateApolloConfig(*aCtx.apolloConfig, aCtx.apolloConfigServer)

	if err := r.reconcileConfig(aCtx); err != nil {
		log.Info("Got error while reconciling, will retry", "err", err.Error())
		return internal.RequeueImmediately()
	}
	return internal.NoRequeue()

}

func (r *ApolloConfigReconciler) reconcileConfig(ctx apolloConfigContext) error {
	if ctx.apolloConfig.Status.SyncStatus != SYNC_SYNCING {
		return nil
	}
	apolloConfig := ctx.apolloConfig

	// Define the ConfigMap
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      apolloConfig.Spec.ConfigMap,
			Namespace: apolloConfig.Namespace,
		},
	}
	configMapData := make(map[string]string)

	releaseKey, content, err := r.ConfigStore.GetRemoteConfig(ctx.apolloConfig)
	if err != nil {
		return r.updateStatusAndReturnError(ctx, SYNC_SYNCING, releaseKey, errors.Wrap(err, "Unable to get config from apollo"))
	}
	fileName := apolloConfig.Spec.FileName
	if len(fileName) == 0 {
		fileName = apolloConfig.Spec.Apollo.NamespaceName
	}
	if len(fileName) == 0 {
		fileName = "application.properties"
	}

	configMapData[fileName] = content

	// Mutate the ConfigMap to match the ApolloConfig Spec
	mutateFn := func() error {
		configMap.Data = configMapData
		return controllerutil.SetControllerReference(apolloConfig, configMap, r.Scheme)
	}
	// compare configmap, if deep equals ignore update
	storedConfigMap := new(corev1.ConfigMap)
	err = r.Get(ctx, types.NamespacedName{Namespace: configMap.Namespace, Name: configMap.Name}, storedConfigMap)
	notFound := false
	if err != nil {
		if apierrs.IsNotFound(err) {
			notFound = true
		} else {
			return nil
		}
	}

	if notFound || !reflect.DeepEqual(configMap.Data, storedConfigMap.Data) {
		// Create or Update the ConfigMap
		if _, err = controllerutil.CreateOrUpdate(ctx, r.Client, configMap, mutateFn); err != nil {
			return r.updateStatusAndReturnError(ctx, SYNC_SYNCING, releaseKey, errors.Wrap(err, "Unable to create or update ConfigMap"))
		} else {
			return r.updateStatusWithAdditional(ctx, SYNC_SUCCESS, releaseKey, "")
		}
	}
	return nil
}

// Helper method to update the status with the error message and status. If there was an error updating the status, return
// that error instead.
func (r *ApolloConfigReconciler) updateStatusAndReturnError(ctx apolloConfigContext, status, releaseKey string, reconcileErr error) error {
	if err := r.updateStatusWithAdditional(ctx, status, releaseKey, reconcileErr.Error()); err != nil {
		return errors.Wrapf(reconcileErr, "Unable to update status with error. Status failure was caused by: '%s'", err.Error())
	}
	return reconcileErr
}

// Update the status and other informational fields. The "additional" parameter should be used to convey additional error information. Leave empty to omit.
// Returns an error if there was a failure to update.
func (r *ApolloConfigReconciler) updateStatusWithAdditional(ctx apolloConfigContext, status, releaseKey, additional string) error {

	log := log2.FromContext(ctx)

	apolloConfigStatus := &ctx.apolloConfig.Status
	apolloConfigStatus.SyncStatus = status
	if releaseKey != "" {
		apolloConfigStatus.ReleaseKey = releaseKey
	}
	if status == SYNC_SYNCING {
		apolloConfigStatus.UpdateAt = metav1.Now()
		apolloConfigStatus.Message = additional
	} else {
		now := metav1.Now()
		apolloConfigStatus.UpdateAt = now
		apolloConfigStatus.LastSyncedSuccess = now
		apolloConfigStatus.Message = additional
	}

	if err := r.Status().Update(ctx, ctx.apolloConfig); err != nil {
		err = errors.Wrap(err, "Unable to update status")
		log.Info("Error while updating status.", "err", err)
		return err
	} else {
		log.Info("Update status", "status", status, "additional", additional)
	}

	return nil
}

// SetupWithManager sets up the apollosync with the Manager.
func (r *ApolloConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&apollov1.ApolloConfig{}).
		Named("apolloconfig").
		WithEventFilter(ApolloConfigChangedPredicate{}).
		Complete(r)
}

var _ predicate.Predicate = ApolloConfigChangedPredicate{}

// ApolloConfigChangedPredicate implements a default update predicate function on annotation change.
type ApolloConfigChangedPredicate struct {
	predicate.TypedFuncs[client.Object]
}

// Update implements default UpdateEvent filter for validating annotation change.
func (ApolloConfigChangedPredicate) Update(e event.TypedUpdateEvent[client.Object]) bool {
	if isNil(e.ObjectOld) {
		return false
	}
	if isNil(e.ObjectNew) {
		return false
	}
	//spec changed
	if e.ObjectNew.GetGeneration() != e.ObjectOld.GetGeneration() {
		return true
	}
	// check if e.ObjectNew type is ApolloConfig, true convert to ApolloConfig
	if apolloConfigNew, ok := e.ObjectNew.(*apollov1.ApolloConfig); ok {
		if apolloConfigOld, ok := e.ObjectOld.(*apollov1.ApolloConfig); ok {

			if apolloConfigNew.Status.NotificationId != apolloConfigOld.Status.NotificationId {
				return true
			}
		}
	}

	return false
}
func isNil(arg any) bool {
	if v := reflect.ValueOf(arg); !v.IsValid() || ((v.Kind() == reflect.Ptr ||
		v.Kind() == reflect.Interface ||
		v.Kind() == reflect.Slice ||
		v.Kind() == reflect.Map ||
		v.Kind() == reflect.Chan ||
		v.Kind() == reflect.Func) && v.IsNil()) {
		return true
	}
	return false
}
