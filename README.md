# drawvclocks: Visualize vector clocks with graphviz

drawvclocks reads a set of vector clocks from a text file and outputs a graph of them in graphviz format. Each clock can optionally have a label. For a detailed description, see [my blog post](http://www.evanjones.ca/visualizing-vector-clocks.html).


## Example input and output

See [`example.txt`](example.txt):

```
a 0 1
b 1 2
c 1 3

x 1 0
y 2 1
z 3 1
```

Output:

![graph of example.txt](example.png)


## Example usage:

1. Install Graphviz. Mac: `brew install graphviz`; Debian/Ubuntu: `apt-get install graphviz`
2. Build drawvclocks: `go build cmd/drawvclocks.go`
3. Run it: `./drawvclocks -format=pdf example.txt > example.pdf`
