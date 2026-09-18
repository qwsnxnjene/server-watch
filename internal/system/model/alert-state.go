package model

// AlertState содержит текущее состояние алерта: счётчик срабатываний,
// условие и признак активного алерта
type AlertState struct {
	Count     int64
	Condition AlertCondition
	Active    bool
}
