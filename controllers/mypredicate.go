package controllers

import (
	"context"
	"fmt"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1  "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/api/core/v1"

	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	userappv1 "user.hjjzs.xyz/api/v1"
)

// containerImage 过滤器
type UserFilter struct {
	r *UserReconciler
	predicate.Funcs
}

func NewUserFilter(r *UserReconciler) *UserFilter {
	return &UserFilter{r, predicate.Funcs{}}
}
func (rl *UserFilter) Delete(e event.DeleteEvent) bool {
	u, ok1 := e.Object.(*userappv1.User)
	if ok1 {
		//  删除rolebinding
		if u.Spec.Role == "admin" {
			rb := rbacv1.ClusterRoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name: u.Name + Domain + "-RB",
				},
			}
			rl.r.Delete(context.TODO(), &rb)

		}else {
			rb := rbacv1.RoleBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      u.Name + Domain + "-RB",
					Namespace: u.Name + Domain,
				},
			}
			rl.r.Delete(context.TODO(), &rb)
		}
		// 删除用于获取镜像的rolebing

		rb2 := rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: "allow-clone-to" + UserName,
				Namespace: "default",
			},
		}
		rl.r.Delete(context.TODO(), &rb2)
		
		// 删除 serviceaccount
		sa := v1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name: u.Name + Domain + "-sa",
				Namespace: u.Name + Domain,
			},
		}
		rl.r.Delete(context.TODO(), &sa)

		// 删除secret 
		se := v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: u.Status.Secret,
				Namespace: u.Name + Domain,
			},
		}
		rl.r.Delete(context.TODO(), &se)
	}
	return false

}

func (rl *UserFilter) Update(e event.UpdateEvent) bool {

	ci, ok1 := e.ObjectOld.(*userappv1.User)
	ci2, ok2 := e.ObjectNew.(*userappv1.User)
	if ok1 && ok2 {
		//
		// if ci.Spec.UserName != ci2.Spec.UserName {
		// 	// fmt.Println("禁止修改")
		// 	ci2.Spec.UserName = ci.Spec.UserName
		// 	rl.r.Update(context.TODO(), ci2)
		// }
		if ci.Spec.Role != ci2.Spec.Role {
			// fmt.Println("禁止修改")
			ci2.Spec.Role = ci.Spec.Role
			rl.r.Update(context.TODO(), ci2)
		}
		if ci.Status.Status == "active" {
			rl.r.UpdataAnnotation(ci2)
			rl.r.Update(context.TODO(), ci2)
		}

		if ci.Status.Status == "" {
			fmt.Println("开始执行controller")
			return true
		} else {
			fmt.Println("不满足执行条件")
			return false
		}
	}

	return false
}
