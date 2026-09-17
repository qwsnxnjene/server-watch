package model

type AlertState struct {
	Count     int64
	Condition AlertCondition
	Active    bool
}
