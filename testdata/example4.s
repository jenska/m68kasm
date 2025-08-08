start:    org $1000
          dc.b "Hi", 0
loop:     dc.w start
          bra loop