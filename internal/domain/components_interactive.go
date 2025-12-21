package domain

// TransitionComponent — указывает, что объект ведет в другое место (лестница, портал).
type TransitionComponent struct {
	TargetLevel int      `json:"targetLevel"`
	TargetPosID EntityID `json:"targetPosID"`
	IsOneWay    bool     `json:"isOneWay"`
}
