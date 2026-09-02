module ChrysoberylKVSP (
    input  wire          clock,
    input  wire          reset,
    input  wire [31:0]   io_romData,
    output wire [9:0]    io_romAddr,
    input  wire [31:0]   io_ramReadData,
    output wire [9:0]    io_ramAddr,
    output wire [31:0]   io_ramWriteData,
    output wire [3:0]    io_ramWriteEnable,
    output wire          io_finishFlag,
    output wire [1023:0] io_x
);
    wire [31:0] x [32];
    wire core_finish;
    reg  finish_latched;

    // Iyokan asserts @reset high; Veryl's reset input is active low.
    chrysoberyl_Chrysoberyl #(
        .ROM_SIZE_BYTES(4096)
    ) core (
        .i_clk(clock),
        .i_rst(~reset),
        .i_rom_data(io_romData),
        .o_rom_addr(io_romAddr),
        .i_ram_read_data(io_ramReadData),
        .o_ram_addr(io_ramAddr),
        .o_ram_write_data(io_ramWriteData),
        .o_ram_we(io_ramWriteEnable),
        .o_finish(core_finish),
        .o_x(x)
    );

    // The shared RISC-V runtime writes a nonzero value to 0x10004. The core's
    // external RAM address contains the low byte-address bits, making offset 4
    // the KVSP completion register.
    always @(posedge clock) begin
        if (reset)
            finish_latched <= 1'b0;
        else if ((|io_ramWriteEnable) && io_ramAddr == 10'd4 && io_ramWriteData != 32'd0)
            finish_latched <= 1'b1;
    end
    assign io_finishFlag = core_finish | finish_latched;

    assign io_x = {
        x[31], x[30], x[29], x[28], x[27], x[26], x[25], x[24],
        x[23], x[22], x[21], x[20], x[19], x[18], x[17], x[16],
        x[15], x[14], x[13], x[12], x[11], x[10], x[9],  x[8],
        x[7],  x[6],  x[5],  x[4],  x[3],  x[2],  x[1],  x[0]
    };
endmodule
