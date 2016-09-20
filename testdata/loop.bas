rem tests looping

10 for i = 0 to 126
20   for j = 0 to 126
30     for k = 0 to 10
40        let a = i * j * k
45        print i; "*"; j; "*";  k; "="; a; "\n"
50     next k
60   next j
70 next i
80 end
