package go_injector

import (
	"fmt"
	"go-injector/inject"
	"testing"
)

type Bean1 struct {
	Name string
}

type Bean2 struct {
	bean1 *Bean1 `autowire:""`
	bean2 *Bean1 `autowire:""`
}

func TestName(t *testing.T) {
	BeanFactory := inject.NewDefaultBeanFactory()
	BeanFactory.RegisterBeanWithName("bean1", &Bean1{Name: "JuST4iT"}).Init(func() {
		fmt.Println("Bean1")
	})
	BeanFactory.RegisterBean(&Bean2{}).Init(func(b *Bean2) {
		fmt.Println("Bean2", b.bean2.Name)
	})

	_ = BeanFactory.AutoWire()
	bean1, _ := BeanFactory.GetBeanByName("bean1")

	bean2, _ := BeanFactory.GetBeanByType((*Bean2)(nil))

	fmt.Println("bean11", bean1.BeanInstance.(*Bean1).Name)
	fmt.Println("bean22", bean2.BeanInstance.(*Bean2).bean2.Name)
}
