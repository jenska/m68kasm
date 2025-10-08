        .org $0000 + 4*4
start:  moveq #5+3-1,d3
        lea (16,a1),a0
        lea (8,pc,d2*2),a1
        bra start
        .byte $AA,$BB,$CC
