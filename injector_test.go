package go_injector

import (
	"fmt"
	"testing"

	"github.com/jeebeys/go-injector/inject"
)

type Bean0 struct {
	Name string
}

type Bean1 struct {
	Name string
}

type Bean2 struct {
	Name string
}

type Bean3 struct {
	bean1  *Bean1            `autowire:"bean1-1"`
	bean2  *Bean1            `autowire:"bean1-2"`
	bean3  *Bean2            `autowire:""`
	beans1 map[string]*Bean1 `autowire:""`
	beans2 []*Bean1          `autowire:""`
}

func TestName(t *testing.T) {
	BeanFactory := inject.NewDefaultBeanFactory()

	//BeanFactory.RegisterBean(Bean0{Name: "bean1-1"}).Init(func(b *Bean1) {
	//	fmt.Println("bean1-1 init", b.Name)
	//})

	BeanFactory.RegisterBeanWithName("bean1-1", &Bean1{Name: "bean1-1"}).Init(func(b *Bean1) {
		fmt.Println("bean1-1 init", b.Name)
	})

	BeanFactory.RegisterBeanWithName("bean1-2", &Bean1{Name: "bean1-2"}).Init(func(b *Bean1) {
		fmt.Println("bean1-2 init", b.Name)
	})

	BeanFactory.RegisterBean(&Bean2{Name: "bean2"}).Init(func(b *Bean2) {
		fmt.Println("bean2 init", b.Name)
	})

	BeanFactory.RegisterBean(&Bean3{}).Init(func(b *Bean3) {
		fmt.Println("Bean3 init", b.bean1.Name)
		fmt.Println("Bean3 init", b.bean2.Name)
		fmt.Println("Bean3 init", b.bean3.Name)
		fmt.Println("Bean3 init", b.beans1)
		fmt.Println("Bean3 init", b.beans2)
	})

	_ = BeanFactory.AutoWire()
	bean1, _ := BeanFactory.GetBeanByName("bean1-1")
	bean2, _ := BeanFactory.GetBeanByType((*Bean3)(nil))
	fmt.Println("bean11 call", bean1.BeanInstance.(*Bean1).Name)
	fmt.Println("bean22 call", bean2.BeanInstance.(*Bean3).bean2.Name)
}
