package ecs

import "iter"

// Seq3 расширяет стандартный пакет iter для трех значений.
// Позволяет писать: for id, a, b := range View2(...)
type Seq3[A, B, C any] func(yield func(A, B, C) bool)

// View1 создает итератор по всем сущностям, имеющим компонент A.
// Аргумент cid — ComponentID для A (для быстрого доступа).
func View1[A any](w *World, cid int) iter.Seq2[EntityID, *A] {
	store := GetStorage[A](w, cid)

	return func(yield func(EntityID, *A) bool) {
		dense := store.values
		ids := store.ids

		// Прямая итерация по массиву (самая быстрая)
		for i := range dense {
			// Передаем EntityID и указатель на значение
			if !yield(ids[i], &dense[i]) {
				return
			}
		}
	}
}

// View2 создает итератор по сущностям, имеющим ОБА компонента (A и B).
// Реализует паттерн "Driver", выбирая кратчайший список для итерации.
func View2[A, B any](w *World, cidA, cidB int) Seq3[EntityID, *A, *B] {
	storeA := GetStorage[A](w, cidA)
	storeB := GetStorage[B](w, cidB)

	return func(yield func(EntityID, *A, *B) bool) {
		lenA := len(storeA.values)
		lenB := len(storeB.values)

		// Выбираем ведущий массив (тот, что меньше)
		if lenA < lenB {
			// A меньше -> бежим по A
			valsA := storeA.values
			idsA := storeA.ids

			for i := range valsA {
				id := idsA[i]

				// Проверяем наличие B (O(1) с ABA проверкой)
				valB := storeB.Get(id)

				if valB != nil {
					if !yield(id, &valsA[i], valB) {
						return
					}
				}
			}
		} else {
			// B меньше -> бежим по B
			valsB := storeB.values
			idsB := storeB.ids

			for i := range valsB {
				id := idsB[i]

				valA := storeA.Get(id)

				if valA != nil {
					// Соблюдаем порядок возврата: A, B
					if !yield(id, valA, &valsB[i]) {
						return
					}
				}
			}
		}
	}
}
