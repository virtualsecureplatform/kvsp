int prime(int index)
{
    if (index <= 0) return 2;

    static int primes[100];
    int nprimes = 0;

    primes[nprimes++] = 2;
    for (int i = 3;; i++) {
        int j = 0;
        while (j < nprimes && i % primes[j] != 0) j++;
        if (j != nprimes) continue;

        // i is a prime.
        if (nprimes == index) return i;
        primes[nprimes++] = i;
    }
}

int main()
{
    return prime(99);
}
