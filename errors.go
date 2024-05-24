package reghive

const (
	ERROR_NO_CHILD = Error("reghive: Child node does not exist")
	ERROR_EXISTS_CHILD = Error("reghive: Child node already exists")
	ERROR_ROOT_MAKE = Error("reghive: Cannot remake root node")
	ERROR_ROOT_DELETE = Error("reghive: Cannot delete root node")
	ERROR_VALUE_TYPE = Error("reghive: Unknown value type")
	ERROR_SEEK_WHENCE = Error("reghive: Invalid whence for seek")
	ERROR_PARENT_ROOT = Error("reghive: Parent not found, trying to GetParent() on root?")
	ERROR_CHILD_MISSING = Error("reghive: Failed to find the requested child")
	ERROR_VALUE_MISSING = Error("reghive: Failed to find the requested value")
	ERROR_BCDDEVICE_HEADER_SIZE = Error("reghive: BCD device must have header size of 0x10 (16)")
)

type Error string
func (e Error) Error() string {
	return string(e)
}