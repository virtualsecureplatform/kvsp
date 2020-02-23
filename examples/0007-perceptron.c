static int weight[3];

static int predict(int x, int y)
{
    return weight[0] * x + weight[1] * y + weight[2] > 0 ? 1 : 0;
}

static void train(int x, int y, int z)
{
    int val = predict(x, y);
    if (val < z) {
        weight[0] += x;
        weight[1] += y;
        weight[2] += z;
    }
    else if (z < val) {
        weight[0] -= x;
        weight[1] -= y;
        weight[2] -= z;
    }
}

int main()
{
    for (int i = 0; i < 10; i++) {
        train(0, 0, 1);
        train(0, 1, 1);
        train(1, 0, 1);
        train(1, 1, 0);
    }

    return predict(0, 0) | (predict(0, 1) << 1) | (predict(1, 0) << 2) |
           (predict(1, 1) << 3);
}
