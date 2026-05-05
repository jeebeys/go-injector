package inject

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
	"unsafe"

	"github.com/jeebeys/go-injector/utils"
)

var NotExistBeanError = errors.New("Not exeis bean")
var AutoWireFinishedError = errors.New("autowire is finish,can not autowire")

type RegisteStatus uint

const (
	Initialize     RegisteStatus = iota //initialize status
	Registeing                          // regeisteing
	InjectFinished                      // Inject finished
)

// Bean
type Bean struct {
	UniqueName   string // bean global unique name
	BeanType     reflect.Type
	BeanValue    reflect.Value
	BeanInstance interface{} // instance
	Alias        string      //bean alias
}

// BeanFactory inteface
type BeanFactory interface {
	RegisterBean(instance interface{}) *DefaultBeanFactory
	RegisterBeanWithName(aliasName string, instance interface{}) *DefaultBeanFactory // registe with alias name
	GetBeanByType(beanType interface{}) (*Bean, error)
	GetBeanByName(beanName string) (*Bean, error) //get bean by name
	Init(fn any)
	CanAutoWire() bool // check if can autowire
	AutoWire() error   // finish bean inject
}

type DefaultBeanFactory struct {
	beanMap       map[string]*Bean // bean map
	funcMap       map[string]any
	beanAliasMap  map[string]string // bean alias bean map
	registeStatus RegisteStatus     //regeiste status
	mutx          sync.RWMutex      //sync.mutx
	bean          *Bean
}

var _ BeanFactory = &DefaultBeanFactory{} //static check

// NewDefaultBeanFactory init method
func NewDefaultBeanFactory() *DefaultBeanFactory {
	return &DefaultBeanFactory{
		beanMap:      make(map[string]*Bean),
		funcMap:      make(map[string]any),
		beanAliasMap: make(map[string]string),
	}
}

// RegisterBean register a bean to BeanFactory
func (defaultBeanFactory *DefaultBeanFactory) RegisterBean(instance interface{}) *DefaultBeanFactory {
	bean := defaultBeanFactory.createBean("", instance)
	defaultBeanFactory.mutx.Lock()
	defer defaultBeanFactory.mutx.Unlock()
	defaultBeanFactory.addToFactory(bean)
	return defaultBeanFactory
}

func (defaultBeanFactory *DefaultBeanFactory) createBean(aliasName string, instance interface{}) *Bean {
	if !utils.CanRegeiste(instance) {
		panic(fmt.Sprintf("%#v is not a ptr", instance))
	}

	defaultBeanFactory.bean = &Bean{}
	defaultBeanFactory.bean.BeanType = reflect.TypeOf(instance)
	defaultBeanFactory.bean.BeanValue = reflect.ValueOf(instance)
	defaultBeanFactory.bean.BeanInstance = instance
	if aliasName != "" {
		defaultBeanFactory.bean.Alias = aliasName
	} else {
		defaultBeanFactory.bean.Alias = defaultBeanFactory.bean.BeanType.String()
	}
	defaultBeanFactory.bean.UniqueName = fmt.Sprintf("%s@%p", defaultBeanFactory.bean.Alias, defaultBeanFactory.bean.BeanInstance)
	return defaultBeanFactory.bean
}

// CanAutoWire check can autowire
func (defaultBeanFactory *DefaultBeanFactory) CanAutoWire() bool {
	if defaultBeanFactory.registeStatus == Initialize {
		return true
	}
	return false
}

// RegisterBeanWithName RegisterBean register
func (defaultBeanFactory *DefaultBeanFactory) RegisterBeanWithName(aliasName string, instance interface{}) *DefaultBeanFactory {
	bean := defaultBeanFactory.createBean(aliasName, instance)
	t := reflect.TypeOf(instance)
	if t.Kind() != reflect.Ptr {
		panic(fmt.Sprintf("inject struct must be ptr.%v", instance))
	}
	defaultBeanFactory.addToFactory(bean)
	return defaultBeanFactory
}
func (defaultBeanFactory *DefaultBeanFactory) addToFactory(bean *Bean) {
	if _, ok := defaultBeanFactory.beanMap[bean.UniqueName]; ok {
		panic(fmt.Sprintf("can not repeat registe bean,alias:%v,instance:%#v", bean.Alias, bean.BeanInstance))
	}
	defaultBeanFactory.beanMap[bean.UniqueName] = bean
	aliasName := bean.Alias
	if aliasName != "" {
		if _, ok := defaultBeanFactory.beanAliasMap[aliasName]; !ok {
			defaultBeanFactory.beanAliasMap[aliasName] = bean.UniqueName
		}
	}
}

// GetBeanByName get bean by name or alias name
func (defaultBeanFactory *DefaultBeanFactory) GetBeanByName(beanName string) (*Bean, error) {
	// alias bean name
	if trueBeanName, ok := defaultBeanFactory.beanAliasMap[beanName]; ok {
		if _, ok := defaultBeanFactory.beanMap[trueBeanName]; ok {
			return defaultBeanFactory.beanMap[trueBeanName], nil
		}
	}
	if _, ok := defaultBeanFactory.beanMap[beanName]; ok {
		return defaultBeanFactory.beanMap[beanName], nil
	}
	return nil, NotExistBeanError
}

func (defaultBeanFactory *DefaultBeanFactory) GetBeanByType(beanType interface{}) (*Bean, error) {
	var bean *Bean
	var count = 0
	for _, val := range defaultBeanFactory.beanMap {
		if val.BeanType == reflect.TypeOf(beanType) {
			bean = val
			count = count + 1
		}
	}
	if count == 0 {
		return nil, NotExistBeanError
	}
	if count == 1 {
		return bean, nil
	}

	panic(fmt.Sprintf("%v is repeat", reflect.TypeOf(beanType)))
}

func (defaultBeanFactory *DefaultBeanFactory) Init(fn any) {
	funcType := reflect.TypeOf(fn)
	if funcType.Kind() != reflect.Func {
		panic(fmt.Sprintf("fn is func(){} or func(*bean){}"))
	}

	if funcType.NumIn() != 0 && funcType.NumIn() != 1 {
		panic(fmt.Sprintf("fn is func(){} or func(*bean){}"))
	}

	defaultBeanFactory.funcMap[defaultBeanFactory.bean.UniqueName] = fn

}

// AutoWire finish to inject
func (defaultBeanFactory *DefaultBeanFactory) AutoWire() error {
	defaultBeanFactory.mutx.Lock()
	defer defaultBeanFactory.mutx.Unlock()
	if !defaultBeanFactory.CanAutoWire() {
		return AutoWireFinishedError
	}

	defaultBeanFactory.registeStatus = Registeing

	for _, val := range defaultBeanFactory.beanMap {
		elemType := val.BeanType.Elem()
		elemVal := val.BeanValue.Elem()
		for i := 0; i < elemType.NumField(); i++ {
			fieldType := elemType.Field(i)
			valueType := elemVal.Field(i)

			tag := fieldType.Tag
			if !utils.FieldNeedToInject(fieldType) {
				continue
			}

			if valueType.Type().Kind() == reflect.Map {
				valueMap := defaultBeanFactory.getInjectMaps(valueType.Type().Elem())
				valuePtr := unsafe.Pointer(valueType.UnsafeAddr())
				reflect.NewAt(valueType.Type(), valuePtr).Elem().Set(valueMap)
			} else if valueType.Type().Kind() == reflect.Slice {
				valueSlice := defaultBeanFactory.getInjectSlice(valueType.Type().Elem())
				valuePtr := unsafe.Pointer(valueType.UnsafeAddr())
				reflect.NewAt(valueType.Type(), valuePtr).Elem().Set(valueSlice)
			} else {
				alias := tag.Get(utils.AutowireTagKey)
				var bean *Bean
				var fullUniqueName string
				if alias != "" {
					fullUniqueName, _ = defaultBeanFactory.beanAliasMap[alias]
				} else {
					fullUniqueName = fmt.Sprintf("%s@%p", valueType.Type().String(), &val.BeanInstance)
				}
				bean, _ = defaultBeanFactory.beanMap[fullUniqueName]
				valuePtr := unsafe.Pointer(valueType.UnsafeAddr())
				reflect.NewAt(valueType.Type(), valuePtr).Elem().Set(bean.BeanValue)
			}
		}
	}

	for key, fn := range defaultBeanFactory.funcMap {
		funcType := reflect.TypeOf(fn)
		_funcVal := reflect.ValueOf(fn)
		//无参数时
		if funcType.NumIn() == 0 {
			_funcVal.Call([]reflect.Value{})
		}

		if funcType.NumIn() == 1 {
			_funcVal.Call([]reflect.Value{reflect.ValueOf(defaultBeanFactory.beanMap[key].BeanInstance)})
		}
	}

	defaultBeanFactory.registeStatus = InjectFinished
	return nil
}

func (defaultBeanFactory *DefaultBeanFactory) getInjectMaps(beanType reflect.Type) reflect.Value {
	mapType := reflect.MapOf(reflect.TypeOf(""), beanType)
	mapValue := reflect.MakeMap(mapType)
	for key, val := range defaultBeanFactory.beanMap {
		if val.BeanType == beanType {
			_key := reflect.ValueOf(key)
			_val := reflect.ValueOf(val.BeanInstance)
			mapValue.SetMapIndex(_key, _val)
		}
	}
	return mapValue
}

func (defaultBeanFactory *DefaultBeanFactory) getInjectSlice(beanType reflect.Type) reflect.Value {
	sliceValue := reflect.MakeSlice(reflect.SliceOf(beanType), 0, 0)
	for _, val := range defaultBeanFactory.beanMap {
		if val.BeanType == beanType {
			sliceValue = reflect.Append(sliceValue, reflect.ValueOf(val.BeanInstance))
		}
	}
	return sliceValue
}

func (defaultBeanFactory *DefaultBeanFactory) string() {
	for k, v := range defaultBeanFactory.beanMap {
		fmt.Printf("k:%#v,v:%#v \n", k, v)
	}
}
