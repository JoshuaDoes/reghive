package reghive

import (
	"fmt"
	"strings"
	"sync"

	"github.com/JoshuaDoes/crunchio"

	hivex "github.com/gabriel-samfira/go-hivex"
)

//Reghive holds a session for a registry hive
type Reghive struct {
	Hive *hivex.Hivex
	Path string
	Root *Key
}

var (
	_ = fmt.Errorf
	hivesLock sync.Mutex
	hives []*Reghive
)

//Open returns a handle to a specified registry hive, cloning a handle for the same path
func Open(path string) (*Reghive, error) {
	for i := 0; i < len(hives); i++ {
		if hives[i].Path == path {
			return hives[i], nil
		}
	}

	hive, err := hivex.NewHivex(path, hivex.WRITE)
	if err != nil {
		return nil, err
	}
	root, err := hive.Root()
	if err != nil {
		return nil, err
	}
	rh := &Reghive{Hive: hive, Path: path}

	key, err := rh.DecodeKey(root)
	if err != nil {
		return nil, err
	}
	rh.Root = key

	if hives == nil {
		hives = make([]*Reghive, 0)
	}
	hives = append(hives, rh)

	return rh, nil
}

//Close closes the handle for the registry hive, nullifying usage of it
func (rh *Reghive) Close() error {
	path := rh.Path

	if err := rh.Hive.Close(); err != nil {
		return err
	}
	rh.Hive = nil
	rh.Path = ""
	rh.Root = nil

	if hives != nil && len(hives) > 0 {
		hivesLock.Lock()
		index := -1
		for i := 0; i < len(hives); i++ {
			if hives[i].Path == path {
				index = i
				break
			}
		}
		if index > -1 {
			hives = append(hives[:index], hives[index+1:]...)
		}
		hivesLock.Unlock()
	}
	return nil
}

func (rh *Reghive) sync() error {
	_, err := rh.Hive.Commit()
	return err
}

//PathSplit splits a key path into an array of nodes by name (without checking against hives, used internally)
func PathSplit(key string) []string {
	if key == "" || key == "/" {
		return make([]string, 0)
	}
	//Node keys are case insensitive, and so will be this search
	key = strings.ToLower(key)
	//Split the requested path into each node key to nest into
	keys := strings.Split(key, "/")
	//Strip path node shenanigans
	if keys[0] == "" { //path starts with /
		keys = keys[1:]
	}
	if keys[len(keys)-1] == "" { //path ends with /
		keys = keys[:len(keys)-1]
	}
	return keys
}

//GetKey returns a specified key (case insensitive)
func (rh *Reghive) GetKey(name string) (*Key, error) {
	keys := PathSplit(name)
	if len(keys) <= 0 {
		return rh.Root, nil
	}

	//Loop over each key in the path, nesting deeper into the node structure to find our target key
	key := rh.Root
	for i := 0; i < len(keys); i++ {
		child, err := key.GetChild(keys[i])
		if err != nil {
			return nil, err
		}
		key = child
	}
	return key, nil
}

//MakeKey makes a specified key if it doesn't exist (including any parent keys) and returns it (case sensitive)
func (rh *Reghive) MakeKey(name string) (*Key, error) {
	keys := PathSplit(name)
	if len(keys) <= 0 {
		return nil, ERROR_ROOT_MAKE
	}

	key := rh.Root
	new := 0
	for i := 0; i < len(keys); i++ {
		new = i
		child, err := key.GetChild(keys[i])
		if err != nil {
			break
		}
		key = child
	}

	for i := new; i < len(keys); i++ {
		child, err := key.MakeChild(keys[i])
		if err != nil {
			return nil, err
		}
		key = child
	}
	return key, nil
}

//DeleteKey removes a specified key and its children (case insensitive)
func (rh *Reghive) DeleteKey(name string) error {
	key, err := rh.GetKey(name)
	if err != nil {
		return err
	}
	keyName, err := key.GetName()
	if err != nil {
		return err
	}
	parent, err := key.GetParent()
	if err != nil {
		return err
	}
	if err := parent.DeleteChild(keyName); err != nil {
		return err
	}
	return rh.sync()
}

//DecodeKey decodes the specified node into a key
func (rh *Reghive) DecodeKey(node int64) (*Key, error) {
	k := &Key{Reghive: rh, Node: node}
	return k, nil
}

//DecodeValue decodes the specified node into a value
func (rh *Reghive) DecodeValue(node, value int64) (*Value, error) {
	k, err := rh.DecodeKey(node)
	if err != nil {
		return nil, err
	}
	v := newValue(k)
	valueKey, err := rh.Hive.NodeValueKey(value)
	if err != nil {
		return nil, err
	}
	if valueKey == "" {
		valueKey = "@"
	}
	v.Name = valueKey
	valueType, valueBytes, err := rh.Hive.ValueValue(value)
	if err != nil {
		return nil, err
	}
	v.Type = RegValueType(valueType)
	v.Write(valueBytes)
	v.ready = true
	return v, nil
}

//Key holds a node (AKA key) from a registry hive
type Key struct {
	Reghive *Reghive
	Node    int64
}

//GetName returns the name of this key
func (k *Key) GetName() (string, error) {
	name, err := k.Reghive.Hive.NodeName(k.Node)
	return name, err
}

//GetParent returns the parent key to this child
func (k *Key) GetParent() (*Key, error) {
	parent, err := k.Reghive.Hive.NodeParent(k.Node)
	if err != nil {
		return nil, err
	}
	return k.Reghive.DecodeKey(parent)
}

//GetChildNames returns a list of all values for this key in order
func (k *Key) GetChildNames() ([]string, error) {
	nodes, err := k.Reghive.Hive.NodeChildren(k.Node)
	if err != nil {
		return nil, err
	}
	children := make([]string, len(nodes))
	for i := 0; i < len(nodes); i++ {
		name, err := k.Reghive.Hive.NodeName(nodes[i])
		if err != nil {
			return nil, err
		}
		children[i] = name
	}
	return children, nil
}

//GetChild returns a specified child key (case insensitive)
func (k *Key) GetChild(name string) (*Key, error) {
	node, err := k.Reghive.Hive.NodeGetChild(k.Node, name)
	if err != nil {
		return nil, err
	}
	if node == 0 {
		return nil, ERROR_CHILD_MISSING
	}
	return k.Reghive.DecodeKey(node)
}

//MakeChild makes a specified child key (case sensitive)
func (k *Key) MakeChild(name string) (*Key, error) {
	node, err := k.Reghive.Hive.NodeAddChild(k.Node, name)
	if err != nil {
		return nil, err
	}
	if err := k.Reghive.sync(); err != nil {
		return nil, err
	}
	return k.Reghive.DecodeKey(node)
}

//DeleteChild recursively removes a specified child key (case insensitive)
func (k *Key) DeleteChild(name string) error {
	node, err := k.Reghive.Hive.NodeGetChild(k.Node, name)
	if err != nil {
		return err
	}
	if _, err := k.Reghive.Hive.NodeDeleteChild(node); err != nil {
		return err
	}
	return k.Reghive.sync()
}

//GetValueNames returns a list of all values for this key in order
func (k *Key) GetValueNames() ([]string, error) {
	nodes, err := k.Reghive.Hive.NodeValues(k.Node)
	if err != nil {
		return nil, err
	}
	values := make([]string, len(nodes))
	for i := 0; i < len(nodes); i++ {
		name, err := k.Reghive.Hive.NodeValueKey(nodes[i])
		if err != nil {
			return nil, err
		}
		values[i] = name
	}
	return values, nil
}

//GetValue finds and returns a value (case insensitive)
func (k *Key) GetValue(name string) (*Value, error) {
	if name == "@" {
		name = ""
	}
	node, err := k.Reghive.Hive.NodeGetValue(k.Node, name)
	if err != nil {
		return nil, err
	}
	return k.Reghive.DecodeValue(k.Node, node)
}

//MakeValue
func (k *Key) MakeValue(name string) (*Value, error) {
	v := newValue(k)
	v.SetName(name)
	v.ready = true
	if err := v.sync(); err != nil {
		return nil, err
	}
	return k.GetValue(name)
}

//DeleteValue deletes a value from a key (case insensitive)
func (k *Key) DeleteValue(name string) error {
	node, err := k.Reghive.Hive.NodeGetValue(k.Node, name)
	if err != nil {
		return err
	}
	values, err := k.Reghive.Hive.NodeValues(k.Node)
	if err != nil {
		return err
	}
	index := -1
	for i := 0; i < len(values); i++ {
		if values[i] == node {
			index = i
			break
		}
	}
	if index > -1 {
		values = append(values[:index], values[index+1:]...)
	}
	hiveValues := make([]hivex.HiveValue, len(values))
	for i := 0; i < len(values); i++ {
		valueKey, err := k.Reghive.Hive.NodeValueKey(values[i])
		if err != nil {
			return err
		}
		valType, valBytes, err := k.Reghive.Hive.ValueValue(values[i])
		if err != nil {
			return err
		}
		hiveValues[i] = hivex.HiveValue{
			Type: int(valType),
			Key: valueKey,
			Value: valBytes,
		}
	}
	if _, err := k.Reghive.Hive.NodeSetValues(k.Node, hiveValues); err != nil {
		return err
	}
	return k.Reghive.sync()
}

//Value holds a value from a key that can be changed or written to
type Value struct {
	*crunchio.Buffer

	Key    *Key
	Name   string
	Type   RegValueType
	ready  bool //true if allowed to sync changes to hive
}

func newValue(k *Key) *Value {
	return &Value{crunchio.NewBuffer(), k, "", RegNone, false}
}

func (v *Value) sync() error {
	if !v.ready {
		return nil
	}
	hv := v.hiveValue()
	if _, err := v.Key.Reghive.Hive.NodeSetValue(v.Key.Node, hv); err != nil {
		return err
	}
	return v.Key.Reghive.sync()
}

func (v *Value) hiveValue() hivex.HiveValue {
	hv := hivex.HiveValue{
		Type: int(v.Type),
		Key: v.Name,
		Value: v.Bytes(),
	}
	if len(hv.Value) == 0 { //Default to REG_NONE with a single byte so hivex can still make a pointer
		hv.Value = make([]byte, 1)
		hv.Type = int(RegNone)
	}
	return hv
}

//SetType changes the type of the value
func (v *Value) SetType(valType RegValueType) error {
	v.Type = valType
	return v.sync()
}

//SetName changes the name of the value
func (v *Value) SetName(name string) error {
	if name == "@" {
		name = ""
	}
	v.Name = name
	return v.sync()
}

//SetValue changes the data of the value using a supported input type
//Data must be one of the following types:
//
// >> TODO: Add registry value types again now that we're using crunchio <<
//
//- BCDDevice (BCD_DEVICE)
//- BCDDescType (BCD_DESCTYPE)
//- byte, bool, int, uint (DWORD_LITTLE)
//- []byte (BINARY)
//- string (SZ)
//- []string (TODO)
//- int16, int32, uint16, uint32 (DWORD_LITTLE)
//- int64, uint64 (QWORD)
//- float32, float64 (BINARY)
//Alternatively, data may satisfy anything supported by crunchio's WriteAbstract method
func (v *Value) SetValue(data interface{}) error {
	switch data.(type) {
	case BCDDevice:
		return ERROR_VALUE_TYPE //TODO: Encode BCDDevice to []byte
	case BCDDescType:
		return ERROR_VALUE_TYPE //TODO: Encode BCDDescType to []byte
	}
	return v.WriteAbstract(data)
}
