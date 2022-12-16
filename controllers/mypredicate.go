package controllers

import (
	"context"
	"fmt"

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

func (rl *UserFilter) Update(e event.UpdateEvent) bool {

	ci, ok1 := e.ObjectOld.(*userappv1.User)
	ci2, ok2 := e.ObjectNew.(*userappv1.User)
	if ok1 && ok2 {
		//
		if ci.Spec.UserName != ci2.Spec.UserName {
			// fmt.Println("禁止修改")
			ci2.Spec.UserName = ci.Spec.UserName
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
