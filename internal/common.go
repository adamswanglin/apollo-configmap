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

package internal

import (
	"strings"
	"time"

	"github.com/pkg/errors"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const SYNC_STATUS_SYNCING = "Syncing"
const SYNC_STATUS_FAIL = "Fail"
const SYNC_STATUS_SUCCESS = "Success"

// RequeueIfError requeues if an error is found.
func RequeueIfError(err error) (ctrl.Result, error) {
	return ctrl.Result{}, err
}

// RequeueImmediatelyUnlessGenerationChanged is a helper function which requeues immediately if the object generation has not changed.
// Otherwise, since the generation change will trigger an immediate update anyways, this
// will not requeue.
// This prevents some cases where two reconciliation loops will occur.
func RequeueImmediatelyUnlessGenerationChanged(prevGeneration, curGeneration int64) (ctrl.Result, error) {
	if prevGeneration == curGeneration {
		return RequeueImmediately()
	}
	return NoRequeue()
}

// Note: In unit test we use Result from apollosync-runtime package from https://github.com/kubernetes-sigs/controller-runtime/blob/2027a413747f2a8cada813dd98b3b1473c253913/pkg/reconcile/reconcile.go#L26
//       The semantic is as follows (result.Requeue)
//       if Requeue is True then always requeue duration can be optional.
//       if Requeue is False and if duration is 0 then NO requeue, but if duration is > 0 then requeue after duration. Its a bit  confusing hence this note.
//       Full implementation is available at https://github.com/kubernetes-sigs/controller-runtime/blob/0fdf465bc21be27b20c5b480a1aced84a3347d43/pkg/internal/controller/controller.go#L235-L287

// NoRequeue does not requeue when Requeue is False and duration is 0.
func NoRequeue() (ctrl.Result, error) {
	return RequeueIfError(nil)
}

// RequeueAfterInterval requeues after a duration when duration > 0 is specified.
func RequeueAfterInterval(interval time.Duration, err error) (ctrl.Result, error) {
	return ctrl.Result{RequeueAfter: interval}, err
}

// RequeueImmediately requeues immediately when Requeue is True and no duration is specified.
func RequeueImmediately() (ctrl.Result, error) {
	return ctrl.Result{Requeue: true}, nil
}

// IgnoreNotFound ignores NotFound errors to prevent reconcile loops.
func IgnoreNotFound(err error) error {
	if apierrs.IsNotFound(err) {
		return nil
	}
	return err
}

func HasDeletionTimestamp(obj metav1.ObjectMeta) bool {
	return !obj.GetDeletionTimestamp().IsZero()
}

// KeyToNamespacedName convert key to namespacedName
func KeyToNamespacedName(namespacedName string) (*types.NamespacedName, error) {
	// unmarshal
	nameAndSpace := strings.Split(namespacedName, string(types.Separator))
	if len(nameAndSpace) != 2 {
		return nil, errors.New("Invalid namespacedName :" + namespacedName)
	}
	return &types.NamespacedName{
		Namespace: nameAndSpace[0],
		Name:      nameAndSpace[1],
	}, nil
}
