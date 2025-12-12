
# WebSocket API Reference

Этот документ описывает протокол взаимодействия между клиентом и сервером игры **Cognitive Dungeon**.

**Адрес для подключения:** `ws://localhost:8080/ws`

## Общий поток взаимодействия (Flow)

1.  **Handshake:** Клиент устанавливает WebSocket-соединение и немедленно отправляет команду `LOGIN`, чтобы "привязать" сессию к игровой сущности.
2.  **Update Loop:** Сервер начинает присылать клиенту сообщения `UPDATE`. Такое сообщение приходит каждый раз, когда наступает ход сущности, которой управляет клиент. Оно содержит полное состояние видимого мира.
3.  **Action:** Когда клиент получает `UPDATE`, где `activeEntityId` совпадает с `myEntityId`, он "разблокирует" интерфейс и позволяет игроку совершить действие.
4.  **Command:** Игрок выполняет действие (например, нажимает кнопку движения), и клиент отправляет на сервер соответствующую команду (`MOVE`, `ATTACK` и т.д.).
5.  **Resolution:** Сервер обрабатывает команду, обновляет состояние мира и переходит к следующему актору в очереди ходов. Цикл повторяется.

---

## ➡️ Клиент -> Сервер (Commands)

Все сообщения от клиента должны быть обернуты в объект `ClientCommand`.

### `ClientCommand` (Основной контейнер)

```json
{
  "action": "ACTION_NAME",
  "token": "entity_id_string",
  "payload": { ... }
}
```
-   `action` (string, **required**): Название действия. Определяет, какой `payload` ожидает сервер.
-   `token` (string, **optional**): ID сущности. **Обязателен только для самой первой команды `LOGIN`**. Для всех последующих команд сервер идентифицирует клиента по самому WebSocket-соединению.
-   `payload` (object, **optional**): JSON-объект с данными, необходимыми для выполнения действия. Его структура зависит от `action`.

### Типы действий и их `payload`

#### `LOGIN`
-   **Описание:** Аутентификация клиента и "завладение" сущностью. Должна быть отправлена **сразу после** установления соединения.
-   **Payload:** Не используется.
-   **Пример:**
    ```json
    { "action": "LOGIN", "token": "hero_1" }
    ```

#### `MOVE`
-   **Описание:** Перемещение сущности на одну клетку.
-   **Payload:** `DirectionPayload`
    -   `dx` (number): Смещение по оси X. Допустимые значения: `-1`, `0`, `1`.
    -   `dy` (number): Смещение по оси Y. Допустимые значения: `-1`, `0`, `1`.
-   **Пример (движение вправо):**
    ```json
    { "action": "MOVE", "payload": { "dx": 1, "dy": 0 } }
    ```

#### `ATTACK`
-   **Описание:** Атака другой сущности.
-   **Payload:** `EntityPayload`
    -   `targetId` (string): ID сущности, которую нужно атаковать.
-   **Пример:**
    ```json
    { "action": "ATTACK", "payload": { "targetId": "e_2" } }
    ```

#### `TALK`
-   **Описание:** Попытка заговорить с другой сущностью.
-   **Payload:** `EntityPayload`
    -   `targetId` (string): ID сущности, с которой нужно поговорить.
-   **Пример:**
    ```json
    { "action": "TALK", "payload": { "targetId": "npc_blacksmith" } }
    ```

#### `INTERACT`
-   **Описание:** Универсальное взаимодействие с объектом в мире (например, лестницей, рычагом, сундуком). Игрок должен находиться на той же клетке, что и объект.
-   **Payload:** `EntityPayload`
    -   `targetId` (string): ID сущности-объекта, с которой нужно взаимодействовать.
-   **Пример (использование лестницы):**
    ```json
    { "action": "INTERACT", "payload": { "targetId": "exit_down_from_0" } }
    ```

#### `WAIT`
-   **Описание:** Пропустить ход.
-   **Payload:** Не используется (`{}` или `null`).
-   **Пример:**
    ```json
    { "action": "WAIT", "payload": {} }
    ```

#### `PICKUP`
-   **Описание:** Подобрать предмет с пола. Игрок должен находиться на одной клетке с предметом или соседней.
-   **Payload:** `ItemPayload`
    -   `itemId` (string): ID предмета на карте.
-   **Пример:**
    ```json
    { "action": "PICKUP", "payload": { "itemId": "item_sword_123" } }
    ```

#### `DROP`
-   **Описание:** Выбросить предмет из инвентаря на пол.
-   **Payload:** `ItemPayload`
    -   `itemId` (string): ID предмета в инвентаре.
    -   `count` (number, **optional**): Количество предметов (если предмет складываемый).
-   **Пример:**
    ```json
    { "action": "DROP", "payload": { "itemId": "item_potion_5", "count": 1 } }
    ```

#### `USE`
-   **Описание:** Использовать предмет (съесть еду, выпить зелье).
-   **Payload:** `ItemPayload`
    -   `itemId` (string): ID предмета в инвентаре.
-   **Пример:**
    ```json
    { "action": "USE", "payload": { "itemId": "item_health_potion" } }
    ```

#### `EQUIP`
-   **Описание:** Надеть предмет (оружие или броню). Предмет перемещается из инвентаря в слот экипировки.
-   **Payload:** `ItemPayload`
    -   `itemId` (string): ID предмета в инвентаре.
-   **Пример:**
    ```json
    { "action": "EQUIP", "payload": { "itemId": "item_iron_sword" } }
    ```

#### `UNEQUIP`
-   **Описание:** Снять экипированный предмет. Предмет перемещается в инвентарь.
-   **Payload:** `ItemPayload`
    -   `itemId` (string): ID предмета в слоте экипировки.
-   **Пример:**
    ```json
    { "action": "UNEQUIP", "payload": { "itemId": "item_iron_sword" } }
    ```

---

## ⬅️ Сервер -> Клиент (Updates)

Сервер отправляет клиенту единственный тип сообщения — `ServerResponse`, который содержит полный снимок игрового состояния.

### `ServerResponse` (Основной контейнер)

```json
{
  "type": "UPDATE",
  "tick": 1250,
  "myEntityId": "hero_1",
  "activeEntityId": "e_2",
  "grid": { ... },
  "map": [ ... ],
  "entities": [ ... ],
  "logs": [ ... ]
}
```
-   `type` (string): Тип сообщения. На данный момент всегда `"UPDATE"`.
-   `tick` (number): Текущее глобальное время в игре.
-   `myEntityId` (string): ID сущности, которой управляет данный клиент.
-   `activeEntityId` (string): ID сущности, чей ход сейчас. **Если `activeEntityId === myEntityId`, фронтенд должен разрешить игроку ввод.**
-   `grid` (`GridMeta`): Объект с метаданными о размере карты.
-   `map` (array of `TileView`): Массив всех видимых и исследорованных клиентом тайлов.
-   `entities` (array of `EntityView`): Массив всех видимых клиентом сущностей.
-   `logs` (array of `LogEntry`): Массив новых игровых сообщений.

### Объекты данных (DTOs)

#### `GridMeta`
-   `w` (number): Ширина карты в тайлах.
-   `h` (number): Высота карты в тайлах.

#### `TileView`
-   `x`, `y` (number): Координаты тайла.
-   `symbol` (string): Символ для отображения (e.g., `.` для пола, `#` для стены).
-   `color` (string): Цвет символа (e.g., `#333333`).
-   `isWall` (boolean): `true`, если тайл является непроходимой стеной.
-   `isVisible` (boolean): `true`, если тайл находится в текущем поле зрения.
-   `isExplored` (boolean): `true`, если сущность когда-либо видела этот тайл (для "тумана войны").

#### `EntityView`
-   `id` (string): Уникальный идентификатор сущности.
-   `type` (string): Тип сущности (`PLAYER`, `ENEMY`, `NPC`, `ITEM`).
-   `name` (string): Имя (e.g., "Герой", "Хитрый Гоблин").
-   `pos` (object): Координаты `{ "x": number, "y": number }`.
-   `render` (object): Данные для отображения `{ "symbol": string, "color": string }`.
-   `stats` (`StatsView`, **optional**): Характеристики сущности.
-   `inventory` (`InventoryView`, **optional**): Инвентарь (виден только владельцу).
-   `equipment` (`EquipmentView`, **optional**): Экипировка (видна только владельцу).

#### `StatsView`
-   `hp`, `maxHp` (number): Текущее и максимальное здоровье.
-   `stamina`, `maxStamina` (number, **optional**): Выносливость.
-   `gold`, `strength` (number, **optional**): Другие характеристики.
-   `isDead` (boolean): `true`, если сущность мертва.

#### `InventoryView`
-   `items` (array of `ItemView`): Список предметов в рюкзаке.
-   `maxSlots` (number): Максимальное количество слотов.
-   `currentWeight` (number): Текущий вес вещей.
-   `maxWeight` (number): Максимальный переносимый вес.

#### `EquipmentView`
-   `weapon` (`ItemView`, **optional**): Предмет в слоте оружия.
-   `armor` (`ItemView`, **optional**): Предмет в слоте брони.

#### `ItemView`
-   `id` (string): ID предмета.
-   `name` (string): Название.
-   `symbol` (string): Символ.
-   `color` (string): Цвет.
-   `category` (string): Категория (`weapon`, `armor`, `potion`, `food`, `misc`).
-   `isStackable` (boolean): Можно ли собирать в стаки.
-   `stackSize` (number): Текущее количество в стаке.
-   `damage` (number, **optional**): Урон (для оружия).
-   `defense` (number, **optional**): Защита (для брони).
-   `weight` (number): Вес предмета.
-   `value` (number): Стоимость.
-   `isSentient` (boolean): Является ли предмет разумным (для диалогов).


#### `LogEntry`
-   `id` (string): Уникальный ID.
-   `text` (string): Текст сообщения.
-   `type` (string): Тип лога для стилизации: `INFO`, `COMBAT`, `SPEECH`, `ERROR`.
-   `timestamp` (number): Время создания сообщения (Unix milliseconds).
