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
	"fmt"
	"regexp"

	v1 "k8s.io/api/core/v1"
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

const encryption_char = "******"

//+kubebuilder:rbac:groups=userapp.hjjzs.xyz,resources=users,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=userapp.hjjzs.xyz,resources=users/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=userapp.hjjzs.xyz,resources=users/finalizers,verbs=update

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

	// 创建ns, 如果ns不存在
	ns := v1.Namespace{}
	domain := "-demo"
	err2 := r.Get(ctx, types.NamespacedName{Name: user.Spec.UserName + domain}, &ns)
	if err2 == nil {
		logger.Info("namespece: " + user.Spec.UserName + domain + "已经存在")
	} else {
		ns.ObjectMeta.Name = user.Spec.UserName + domain
		err3 := r.Create(ctx, &ns)
		if err3 != nil {
			logger.Error(err3, "创建ns失败")
		}
	}

	// 修改密码
	secret := v1.Secret{}
	if user.Status.Status != "" {
		if user.Spec.Newpassword == "" {
			user.Status.Message = "修改密码newpassword不能为空"
			r.Status().Update(ctx, &user)
			return ctrl.Result{}, nil
		}
		err3 := r.Get(ctx, types.NamespacedName{Namespace: req.Namespace, Name: user.Status.Secret}, &secret)
		if err3 != nil {
			logger.Error(err3, "无法获取,secret")
		}
		b := secret.Data["md5"]
		s := util.MD5(user.Spec.Password)
		if string(b) != s {
			logger.Error(fmt.Errorf("修改密码失败"), req.Name)
			user.Status.Message = "密码不正确,请将password字段修改为旧密码,当旧密码认证通过后才会以新密码修改。"
			r.Status().Update(ctx, &user)
			user.Spec.Password = encryption_char
			r.Update(ctx, &user)
		} else {
			secret.Data = map[string][]byte{"md5": []byte(util.MD5(user.Spec.Newpassword))}
			r.Update(ctx, &secret)
			user.Spec.Newpassword = ""
			user.Spec.Password = encryption_char
			r.Update(ctx, &user)
			user.Status.Message = "密码修改成功"
			r.Status().Update(ctx, &user)
		}
		return ctrl.Result{}, nil
	}

	// 加密密码并生成secret
	var md5_data string
	if user.Spec.Newpassword != "" {
		md5_data = util.MD5(user.Spec.Newpassword)
	} else {
		md5_data = util.MD5(user.Spec.Password)
	}
	secret2 := v1.Secret{}
	var secretName = user.Spec.UserName + domain + "-secret"
	secret2.ObjectMeta.Name = secretName
	secret2.ObjectMeta.Namespace = req.Namespace
	bb := true
	secret2.ObjectMeta.OwnerReferences = append(secret2.ObjectMeta.OwnerReferences, metav1.OwnerReference{
		APIVersion:         user.APIVersion,
		Kind:               user.Kind,
		Name:               user.Name,
		UID:                user.UID,
		BlockOwnerDeletion: &bb,
	},
	)
	secret2.Data = map[string][]byte{"md5": []byte(md5_data)}
	r.Create(ctx, &secret2)

	// updata user
	user.Spec.Password = encryption_char
	r.UpdataAnnotation(&user)
	// logger.Info(user.Annotations["kubectl.kubernetes.io/last-applied-configuration"])
	r.Update(ctx, &user)
	user.Status.Status = "active"
	user.Status.Secret = secretName
	r.Status().Update(ctx, &user)


	logger.Info("okk")
	return ctrl.Result{}, nil
}

func (r *UserReconciler) UpdataAnnotation(u *userappv1.User) {
	s := u.Annotations["kubectl.kubernetes.io/last-applied-configuration"]
	reg := regexp.MustCompile(`"password":".*?"`)
	s2 := reg.ReplaceAllString(s, `"password":"******"`)
	u.Annotations["kubectl.kubernetes.io/last-applied-configuration"] = s2
}

// SetupWithManager sets up the controller with the Manager.
func (r *UserReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&userappv1.User{}).
		WithEventFilter(NewUserFilter(r)).
		Complete(r)
}
