start:  org $1000
        dc.b "Hi", 0
val:    dc.w $1234, start+2
        align 8
        ds.b 4
        even
loop:   move.b #1, d0
        bra loop