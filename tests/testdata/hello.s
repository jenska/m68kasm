        ; Demo program for m68kasm v0.1.0
        ; Includes .byte, .word, .long, .align directives

        .org $0000 + 4*4

start:  moveq #5+3-1,d3
        lea (16,a1),a0
        lea (8,pc,d2),a1
        bra start

        ; some raw data bytes
        .byte $AA,$BB,$CC

        ; 16-bit words (big-endian)
        .word 1,2,3,4

        ; align to next 4-byte boundary, fill with $CC
        .align 4,$CC

        ; 32-bit longs (big-endian)
        .long $11223344, $AABBCCDD, -1

        ; align to 8-byte boundary (default fill=0)
        .align 8

        ; trailing marker
        .byte $EE,$FF
