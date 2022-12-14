package controllers

import (
	"context"
	"fmt"

	userappv1 "hjjzs.xyz/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
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
		fmt.Println("过滤中")
		// 
		if ci.Spec.UserName != ci2.Spec.UserName {
			// fmt.Println("禁止修改")
			ci2.Spec.UserName = ci.Spec.UserName
			rl.r.Update(context.TODO(), ci2)
		}
		return ci2.Spec.Password != encryption_char
	}

	return false
}
