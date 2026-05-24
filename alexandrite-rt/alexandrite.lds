OUTPUT_ARCH("riscv")

PHDRS
{
    rom PT_LOAD;
    ram PT_LOAD;
}

ENTRY(_start)

SECTIONS
{
    . = 0x00000000;
    .text : {
        *(.text.start)
        *(.text .text.*)
    } :rom

    . = 0x0001000c;
    .data : { *(.data .data.*) } :ram
    .sdata : { *(.sdata .sdata.*) } :ram
    .rodata : { *(.rodata .rodata.*) } :ram
    .bss : {
        *(.sbss .sbss.*)
        *(.bss .bss.*)
        *(COMMON)
    } :ram
}
