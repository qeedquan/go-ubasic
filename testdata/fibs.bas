rem outputs some sequence of the fibonacci

20 let a = 1
40 let b = 1

60 for i = 0 to 20
80 let b = a + b
100 let a = b - a
120 print i; ":", a, b; "\n"
140 next i

160 end
