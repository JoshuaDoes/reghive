package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/LinusGates/reghive"
)

func main() {
	hive := os.Args[1]
	fmt.Println("Opening", hive)
	rh, err := reghive.Open(hive)
	fatal(err, "Failed to open '%s'", hive)
	defer rh.Close()
	fmt.Println("")

	fmt.Println("Getting key /")
	keyParent, err := rh.GetKey("/")
	fatal(err, "Failed to get key")
	printKey(keyParent)
	fmt.Println("Making key '/foo/'")
	keyChild, err := rh.MakeKey("/foo/")
	fatal(err, "Failed to make key '/foo/'")
	fmt.Println("Making value 'bar'")
	val, err := keyChild.MakeValue("bar")
	fatal(err, "Failed to make value 'bar'")
	fmt.Println("Setting value 'bar' to 'asdf'")
	err = val.SetValue("asdf")
	fatal(err, "Failed to set value 'bar' to 'asdf'")
	printKey(keyParent)
	fmt.Println("")

	fmt.Println("Getting key '/foo/'")
	keyChild, err = rh.GetKey("/foo/")
	fatal(err, "Failed to get key '/foo/'")
	printKey(keyChild)
	fmt.Println("Getting value 'bar'")
	val, err = keyChild.GetValue("bar")
	fatal(err, "Failed to get value 'bar'")
	fmt.Println("Value:", val)
}

func printKey(key *reghive.Key) {
	name, err := key.GetName()
	fatal(err, "Failed to get name of key")
	children, err := key.GetChildNames()
	fatal(err, "Failed to get child keys")
	values, err := key.GetValueNames()
	fatal(err, "Failed to get values")
	fmt.Printf("Name: %s\nChildren (%d): %s\nValues (%d): %s\n", name, len(children), strings.Join(children, "\n\t"), len(values), strings.Join(values, "\n\t"))
}

func fatal(err error, format string, args ...any) {
	if err != nil {
		msg := fmt.Sprintf(format, args...)
		msg = fmt.Sprintf("%s: %v", msg, err)
		panic(msg)
	}
}