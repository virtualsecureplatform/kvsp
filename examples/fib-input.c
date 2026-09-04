static int fib(int n) {
  int a = 0;
  int b = 1;
  for (int i = 0; i < n; i++) {
    int next = a + b;
    a = b;
    b = next;
  }
  return a;
}

int main(int argc, char **argv) {
  return fib(argv[1][0] - '0');
}
