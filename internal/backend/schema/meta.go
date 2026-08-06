package schema

type Storage int

const (
	StorageStored Storage = iota
	StorageVirtual
)

type Order int

const (
	OrderDescending Order = -1
	OrderNone       Order = 0
	OrderAscending  Order = 1
)

type ConflictAction int

const (
	ConflictActionNone ConflictAction = iota
	ConflictActionRollback
	ConflictActionAbort
	ConflictActionFail
	ConflictActionIgnore
	ConflictActionReplace
)
