#pragma endian little

struct ReplayFileHeader {
    char magic[4];
    u32 version;
    s64 seed;
    s64 timestamp;
    s32 level_id;
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

ReplayFileHeader file_header @ 0x00;
Action actions[file_header.action_count] @ sizeof(file_header);