/*
Copyright 2022.

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

package controllers

import (
	"context"
	"regexp"
	"time"

	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	userappv1 "user.hjjzs.xyz/api/v1"
	util "user.hjjzs.xyz/utils"
)

// UserReconciler reconciles a User object
type UserReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

var (
	rolename      string
	Domain        = "-demo"
	userNamespace string
)

//+kubebuilder:rbac:groups=userapp.hjjzs.xyz,resources=users,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=userapp.hjjzs.xyz,resources=users/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=userapp.hjjzs.xyz,resources=users/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the User object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *UserReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// TODO(user): your logic here
	user := userappv1.User{}
	err := r.Get(ctx, req.NamespacedName, &user)
	if err != nil {
		logger.Info("删除成功")
		return ctrl.Result{}, nil
	}
	if user.Status.Status == "" {
		// 创建ns, 如果ns不存在
		ns := v1.Namespace{}
		userNamespace = user.Spec.UserName + Domain
		err2 := r.Get(ctx, types.NamespacedName{Name: userNamespace}, &ns)
		if err2 == nil {
			logger.Info("namespece: " + userNamespace + "已经存在")
		} else {
			ns.ObjectMeta.Name = userNamespace
			err3 := r.Create(ctx, &ns)
			if err3 != nil {
				logger.Error(err3, "创建ns失败")
			}
		}

		time.Sleep(time.Second * 2)

		// 权限设置
		// 生成serviceacount
		sa := v1.ServiceAccount{}
		sa.Name = user.Spec.UserName + Domain + "-sa"
		sa.Namespace = user.Spec.UserName + Domain

		err3 := r.Create(ctx, &sa)
		if err3 != nil {
			logger.Error(err3, "sa false")
		}
		// 创建rolebind  START

		// //判断用户role
		if user.Spec.Role == "admin" {
			rolename = "userapp-admin-role"
		} else {
			rolename = "userapp-user-role"
		}

		bind := rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      user.Spec.UserName + Domain + "-RB",
				Namespace: userNamespace,
			},
			RoleRef: rbacv1.RoleRef{
				Name:     rolename,
				Kind:     "ClusterRole",
				APIGroup: "rbac.authorization.k8s.io",
			},
			Subjects: []rbacv1.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      user.Spec.UserName + Domain + "-sa",
					Namespace: userNamespace,
				},
			},
		}
		err4 := r.Create(ctx, &bind)
		if err4 != nil {
			logger.Error(err4, "rbac false")
		}
		// 创建rolebind  END

		// 加密密码并生成secret
		md5_data := util.MD5(user.Spec.Password)

		secret2 := v1.Secret{}
		var secretName = user.Spec.UserName + Domain + "-secret"
		secret2.ObjectMeta.Name = secretName
		secret2.ObjectMeta.Namespace = userNamespace
		secret2.Data = map[string][]byte{"md5": []byte(md5_data)}
		err5 := r.Create(ctx, &secret2)
		if err5 != nil {
			logger.Error(err4, "secret false")
		}

		// updata user
		r.UpdataAnnotation(&user)
		r.Update(ctx, &user)
		user.Status.Status = "active"
		user.Status.Secret = secretName
		r.Status().Update(ctx, &user)

		logger.Info("controller 执行结束")
	}
	return ctrl.Result{}, nil

}

func (r *UserReconciler) UpdataAnnotation(u *userappv1.User) {
	s := u.Annotations["kubectl.kubernetes.io/last-applied-configuration"]
	reg := regexp.MustCompile(`"password":".*?"`)
	s2 := reg.ReplaceAllString(s, `"password":"******"`)
	u.Annotations["kubectl.kubernetes.io/last-applied-configuration"] = s2
	u.Spec.Password = "******"
}

// SetupWithManager sets up the controller with the Manager.
func (r *UserReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&userappv1.User{}).
		WithEventFilter(NewUserFilter(r)).
		Complete(r)
}
