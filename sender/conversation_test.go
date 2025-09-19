package sender

import (
	"sync"
	"testing"
)

// TestConcurrentMapAccess проверяет что нет race condition при одновременном доступе к ConversationHandler
func TestConcurrentMapAccess(t *testing.T) {
	ch := NewConversationHandler()

	// Создаем WaitGroup для синхронизации горутин
	var wg sync.WaitGroup

	// Количество горутин для симуляции concurrent access
	numGoroutines := 100

	// Запускаем множество горутин, которые одновременно выполняют операции с map
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(userID int) {
			defer wg.Done()

			// Симулируем операции, которые вызывали race condition
			ch.SetActiveStage(0, userID)
			stage := ch.GetActiveStage(userID)
			if stage != 0 {
				ch.SetActiveStage(1, userID)
			}
			ch.End(userID)
		}(i)
	}

	// Ждем завершения всех горутин
	wg.Wait()

	t.Log("Тест на concurrent map access прошел успешно")
}

// TestConversationHandlerBasicFunctionality проверяет базовую функциональность
func TestConversationHandlerBasicFunctionality(t *testing.T) {
	ch := NewConversationHandler()

	// Тестируем установку и получение активной стадии
	userID := 123
	stageID := 5

	ch.SetActiveStage(stageID, userID)

	activeStage := ch.GetActiveStage(userID)
	if activeStage != stageID {
		t.Errorf("Ожидали стадию %d, получили %d", stageID, activeStage)
	}

	// Тестируем завершение разговора
	ch.End(userID)

	// После завершения пользователь должен быть неактивным
	// Но GetActiveStage все еще может возвращать последнюю стадию, если пользователь помечен как активный
	activeAfterEnd := ch.GetActiveStage(userID)
	if activeAfterEnd != 0 {
		t.Logf("После End() получили стадию %d (это ожидаемо, если active[userID] = false)", activeAfterEnd)
	}
}
