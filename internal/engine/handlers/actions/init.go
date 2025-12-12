package actions

import "cognitive-server/internal/engine/handlers"

func init() {
	// Assuming actions.RegisterHandler and the Handle* functions are defined elsewhere or will be defined.
	// This block registers various game actions, including inventory management.
	// The specific Handle* functions (e.g., HandleMove, HandlePickup) are not provided in this snippet,
	// but are expected to exist in the 'actions' package.
	// General actions
	// actions.RegisterHandler("MOVE", HandleMove) // Uncomment and define HandleMove if needed
	// actions.RegisterHandler("ATTACK", HandleAttack) // Uncomment and define HandleAttack if needed
	// actions.RegisterHandler("WAIT", HandleWait) // Uncomment and define HandleWait if needed
	// actions.RegisterHandler("TALK", HandleTalk) // Uncomment and define HandleTalk if needed
	// actions.RegisterHandler("INTERACT", HandleInteract) // Uncomment and define HandleInteract if needed

	// Inventory handlers
	// actions.RegisterHandler("PICKUP", HandlePickup) // Uncomment and define HandlePickup if needed
	// actions.RegisterHandler("DROP", HandleDrop) // Uncomment and define HandleDrop if needed
	// actions.RegisterHandler("USE", HandleUse) // Uncomment and define HandleUse if needed
	// actions.RegisterHandler("EQUIP", HandleEquip) // Uncomment and define HandleEquip if needed
	// actions.RegisterHandler("UNEQUIP", HandleUnequip) // Uncomment and define HandleUnequip if needed
}

func HandleInit(ctx handlers.Context) (handlers.Result, error) {
	return handlers.Result{
		Msg:     "Добро пожаловать в Cognitive Dungeon.",
		MsgType: "INFO",
	}, nil
}
