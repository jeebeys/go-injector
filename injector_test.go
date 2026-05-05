package go_injector

import (
	"fmt"
	"testing"

	"github.com/jeebeys/go-injector/inject"
)

type Bean1 struct {
	Name string
}

type Bean2 struct {
	bean1  *Bean1            `autowire:"bean11"`
	bean2  *Bean1            `autowire:"bean12"`
	beans1 map[string]*Bean1 `autowire:""`
	beans2 []*Bean1          `autowire:""`
}

func TestName(t *testing.T) {
	BeanFactory := inject.NewDefaultBeanFactory()

	BeanFactory.RegisterBeanWithName("bean12", &Bean1{Name: "bean12"}).Init(func() {
		fmt.Println("bean12 init")
	})
	BeanFactory.RegisterBeanWithName("bean11", &Bean1{Name: "bean11"}).Init(func() {
		fmt.Println("bean11 init")
	})
	BeanFactory.RegisterBean(&Bean2{}).Init(func(b *Bean2) {
		fmt.Println("Bean2 init", b.bean2.Name)
		fmt.Println("Bean2 init", b.beans1)
		fmt.Println("Bean2 init", b.beans2)
	})

	_ = BeanFactory.AutoWire()
	bean1, _ := BeanFactory.GetBeanByName("bean11")
	bean2, _ := BeanFactory.GetBeanByType((*Bean2)(nil))
	fmt.Println("bean11 call", bean1.BeanInstance.(*Bean1).Name)
	fmt.Println("bean22 call", bean2.BeanInstance.(*Bean2).bean2.Name)
}
