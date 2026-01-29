#pragma endian little

struct ReplayFileHeader {
    char magic[4];
    u32 version;
    s64 seed;
    s64 timestamp;
    s32 level_id;
    u32 player_state_len;
    s32 action_count;
};

struct ActionHeader {
    s32 tick;
    u8 action_type;
    u8 token_len;
    u16 payload_len;
};

struct Action {
    ActionHeader header;
    char token[header.token_len];
    u8 payload[header.payload_len];
};

// 1. Читаем заголовок
ReplayFileHeader header @ 0x00;

// 2. Читаем снапшот
// Если player_state_len == 0, массив будет пустым и не займет места.
char player_snapshot[header.player_state_len] @ sizeof(header);

// 3. Читаем действия
// Смещение = Размер заголовка (36) + Длина снапшота
Action actions[header.action_count] @ sizeof(header) + header.player_state_len;