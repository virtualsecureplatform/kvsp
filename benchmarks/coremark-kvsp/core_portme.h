#ifndef CORE_PORTME_H
#define CORE_PORTME_H

#define HAS_FLOAT 0
#define HAS_TIME_H 0
#define USE_CLOCK 0
#define HAS_STDIO 0
#define HAS_PRINTF 0
#define SEED_METHOD SEED_ARG
#define MEM_METHOD MEM_STATIC
#define MULTITHREAD 1
#define MAIN_HAS_NOARGC 0
#define MAIN_HAS_NORETURN 0
#define COMPILER_VERSION "KVSP clang"
#define COMPILER_FLAGS "-Oz"
#define MEM_LOCATION "STATIC"

typedef signed short ee_s16;
typedef unsigned short ee_u16;
#if __SIZEOF_INT__ == 4
typedef signed int ee_s32;
typedef unsigned int ee_u32;
#else
typedef signed long ee_s32;
typedef unsigned long ee_u32;
#endif
typedef unsigned char ee_u8;
typedef ee_u32 ee_f32;
typedef __UINTPTR_TYPE__ ee_ptr_int;
typedef __SIZE_TYPE__ ee_size_t;
typedef ee_u32 CORE_TICKS;

#define align_mem(x) (void *)(4 + (((ee_ptr_int)(x) - 1) & ~3))
#define NULL ((void *)0)

typedef struct CORE_PORTABLE_S {
  ee_u8 portable_id;
} core_portable;

extern ee_u32 default_num_contexts;
int ee_printf(const char *fmt, ...);

#endif
