package inject

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
	"unsafe"
)

var NotExistBeanError = errors.New("not exist bean")
var AutoWireFinishedError = errors.New("autowire is finish,can not autowire")

type Status uint

const (
	AutowireTagKey = "autowire"
)
const (
	Initialize  Status = iota //initialize status
	Registering               // Registering
	Finished                  // Inject finished
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
	beanMap      map[string]*Bean // bean map
	funcMap      map[string]any
	beanAliasMap map[string]string // bean alias bean map
	status       Status            // Autowire status
	mutex        sync.RWMutex      // sync.mutx
	bean         *Bean
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

func (defaultBeanFactory *DefaultBeanFactory) createBean(aliasName string, instance interface{}) *Bean {
	t := reflect.TypeOf(instance)
	if !(t.Kind() == reflect.Ptr && t.Elem().Kind() == reflect.Struct) {
		panic(fmt.Sprintf("%#v is not a ptr", instance))
	}
	//isStruct := reflect.TypeOf(instance).Kind() == reflect.Struct
	defaultBeanFactory.bean = &Bean{}
	defaultBeanFactory.bean.BeanType = reflect.TypeOf(instance)
	defaultBeanFactory.bean.BeanValue = reflect.ValueOf(instance)
	defaultBeanFactory.bean.BeanInstance = instance

	if aliasName != "" {
		defaultBeanFactory.bean.Alias = aliasName
		defaultBeanFactory.bean.UniqueName = fmt.Sprintf("%s@%p", defaultBeanFactory.bean.Alias, defaultBeanFactory.bean.BeanInstance)

	} else {
		defaultBeanFactory.bean.Alias = fmt.Sprintf("%s_%p", defaultBeanFactory.bean.BeanType.String(), defaultBeanFactory.bean.BeanInstance)
		defaultBeanFactory.bean.UniqueName = fmt.Sprintf("%s@%p", defaultBeanFactory.bean.BeanType.String(), defaultBeanFactory.bean.BeanInstance)
	}
	return defaultBeanFactory.bean
}

// CanAutoWire check can autowire
func (defaultBeanFactory *DefaultBeanFactory) CanAutoWire() bool {
	if defaultBeanFactory.status == Initialize {
		return true
	}
	return false
}

// RegisterBean register a bean to BeanFactory
func (defaultBeanFactory *DefaultBeanFactory) RegisterBean(instance interface{}) *DefaultBeanFactory {
	return defaultBeanFactory.RegisterBeanWithName("", instance)
}

// RegisterBeanWithName RegisterBean register
func (defaultBeanFactory *DefaultBeanFactory) RegisterBeanWithName(aliasName string, instance interface{}) *DefaultBeanFactory {
	bean := defaultBeanFactory.createBean(aliasName, instance)
	defaultBeanFactory.mutex.Lock()
	defer defaultBeanFactory.mutex.Unlock()
	defaultBeanFactory.addToFactory(bean)
	return defaultBeanFactory
}
func (defaultBeanFactory *DefaultBeanFactory) addToFactory(bean *Bean) {
	if _, ok := defaultBeanFactory.beanAliasMap[bean.Alias]; ok {
		panic(fmt.Sprintf("can not repeat registe bean,alias:%v,instance:%#v", bean.Alias, bean.BeanInstance))
	}

	defaultBeanFactory.beanAliasMap[bean.Alias] = bean.UniqueName
	defaultBeanFactory.beanMap[bean.UniqueName] = bean
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
	defaultBeanFactory.mutex.Lock()
	defer defaultBeanFactory.mutex.Unlock()
	if !defaultBeanFactory.CanAutoWire() {
		return AutoWireFinishedError
	}

	defaultBeanFactory.status = Registering
	for _, val := range defaultBeanFactory.beanMap {

		elemType := val.BeanType.Elem()
		elemVal := val.BeanValue.Elem()
		for i := 0; i < elemType.NumField(); i++ {
			fieldType := elemType.Field(i)
			valueType := elemVal.Field(i)

			tag := fieldType.Tag
			if _, ok := tag.Lookup(AutowireTagKey); !ok {
				continue
			}

			if valueType.Type().Kind() == reflect.Map {
				valueMap := defaultBeanFactory.getBeansForMaps(valueType.Type().Elem())
				valuePtr := unsafe.Pointer(valueType.UnsafeAddr())
				reflect.NewAt(valueType.Type(), valuePtr).Elem().Set(valueMap)
			} else if valueType.Type().Kind() == reflect.Slice {
				valueSlice := defaultBeanFactory.getBeansForSlice(valueType.Type().Elem())
				valuePtr := unsafe.Pointer(valueType.UnsafeAddr())
				reflect.NewAt(valueType.Type(), valuePtr).Elem().Set(valueSlice)
			} else if valueType.Type().Kind() == reflect.Ptr {
				aliasName := tag.Get(AutowireTagKey)
				var value reflect.Value
				if aliasName != "" {
					value = defaultBeanFactory.getBeanForName(aliasName, elemType.Name()+"."+fieldType.Name)
				} else {
					value = defaultBeanFactory.getBeanForType(valueType.Type(), elemType.Name()+"."+fieldType.Name)
				}
				if value.IsValid() {
					valuePtr := unsafe.Pointer(valueType.UnsafeAddr())
					reflect.NewAt(valueType.Type(), valuePtr).Elem().Set(value)
				}
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

	defaultBeanFactory.status = Finished
	return nil
}

func (defaultBeanFactory *DefaultBeanFactory) getBeanForName(beanName string, fieldName string) reflect.Value {
	if UniqueName, ok := defaultBeanFactory.beanAliasMap[beanName]; ok {
		if bean, ok := defaultBeanFactory.beanMap[UniqueName]; ok {
			return reflect.ValueOf(bean.BeanInstance)
		}
	}
	panic(fmt.Sprintf("Field %s Inject Error: %s is not exist", fieldName, beanName))
}

func (defaultBeanFactory *DefaultBeanFactory) getBeanForType(beanType reflect.Type, fieldName string) reflect.Value {
	var value reflect.Value
	var count = 0

	for _, val := range defaultBeanFactory.beanMap {
		if val.BeanType == beanType {
			value = reflect.ValueOf(val.BeanInstance)
			count = count + 1
		}
	}
	if count == 0 {
		panic(fmt.Sprintf("Field %s Inject Error: %v is not exist", fieldName, beanType))
	}

	if count != 1 {
		panic(fmt.Sprintf("Field %s Inject Error: %v is more than one", fieldName, beanType))
	}

	return value
}

func (defaultBeanFactory *DefaultBeanFactory) getBeansForMaps(beanType reflect.Type) reflect.Value {
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

func (defaultBeanFactory *DefaultBeanFactory) getBeansForSlice(beanType reflect.Type) reflect.Value {
	sliceValue := reflect.MakeSlice(reflect.SliceOf(beanType), 0, 0)
	for _, val := range defaultBeanFactory.beanMap {
		if val.BeanType == beanType {
			sliceValue = reflect.Append(sliceValue, reflect.ValueOf(val.BeanInstance))
		}
	}
	return sliceValue
}

func (defaultBeanFactory *DefaultBeanFactory) String() {
	for k, v := range defaultBeanFactory.beanMap {
		fmt.Printf("k:%#v,v:%#v \n", k, v)
	}
}
