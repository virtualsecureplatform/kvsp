#include <tfhe/tfhe.h>
#include <tfhe/tfhe_io.h>

#include <cassert>
#include <fstream>
#include <memory>
#include <random>

std::shared_ptr<TFheGateBootstrappingParameterSet> make_param_set(
    TFheGateBootstrappingParameterSet* src)
{
    return std::shared_ptr<TFheGateBootstrappingParameterSet>{
        src, delete_gate_bootstrapping_parameters};
}

std::shared_ptr<TFheGateBootstrappingSecretKeySet> make_secret_key(
    TFheGateBootstrappingSecretKeySet* src)
{
    return std::shared_ptr<TFheGateBootstrappingSecretKeySet>{
        src, delete_gate_bootstrapping_secret_keyset};
}

std::shared_ptr<LweSample> make_lwe_sample(
    const std::shared_ptr<TFheGateBootstrappingSecretKeySet>& key)
{
    return std::shared_ptr<LweSample>{
        new_gate_bootstrapping_ciphertext(key->params),
        delete_gate_bootstrapping_ciphertext};
}

void dump_key(const std::shared_ptr<TFheGateBootstrappingSecretKeySet>& key,
              const std::string& filepath)
{
    std::ofstream ofs{filepath, std::ios_base::binary};
    assert(ofs && "Invalid filepath, maybe you don't have right permission?");
    export_tfheGateBootstrappingSecretKeySet_toStream(ofs, key.get());
}

std::shared_ptr<TFheGateBootstrappingSecretKeySet> import_secret_key(
    const std::string& filepath)
{
    std::ifstream ifs{filepath, std::ios_base::binary};
    assert(ifs && "Invalid filepath, maybe not exists?");
    return make_secret_key(
        new_tfheGateBootstrappingSecretKeySet_fromStream(ifs));
}

void doGenkey(const std::string& output_filepath)
{
    // generate a keyset
    const int minimum_lambda = 110;
    auto params = make_param_set(
        new_default_gate_bootstrapping_parameters(minimum_lambda));

    // generate a random key
    uint32_t seed[] = {std::random_device{}()};
    tfhe_random_generator_setSeed(seed, 1);
    auto key = make_secret_key(
        new_random_gate_bootstrapping_secret_keyset(params.get()));

    // Dump the key.
    dump_key(key, output_filepath);
}

void doCloudkey(const std::string& input_filepath,
                const std::string& output_filepath)
{
    auto secret_key = import_secret_key(input_filepath);

    std::ofstream ofs{output_filepath, std::ios_base::binary};
    assert(ofs && "Invalid filepath, maybe you don't have right permission?");
    export_tfheGateBootstrappingCloudKeySet_toStream(ofs, &secret_key->cloud);
}

void doEnc(const std::string& key_filepath, const std::string& input_filepath,
           const std::string& output_filepath, const std::string& nbits_str)
{
    // nbits may be negative, which means 'no limit about the number of bits'
    auto nbits = std::stoll(nbits_str);

    auto key = import_secret_key(key_filepath);

    std::ifstream ifs{input_filepath, std::ios_base::binary};
    assert(ifs && "Invalid filepath, maybe not exists?");
    std::ofstream ofs{output_filepath, std::ios_base::binary};
    assert(ofs && "Invalid filepath, maybe you don't have right permission?");

    auto value = make_lwe_sample(key);
    while (nbits != 0) {
        unsigned int byte = ifs.get();
        if (byte == EOF) break;
        for (int i = 0; i < 8; i++, byte >>= 1) {
            bootsSymEncrypt(value.get(), byte & 1, key.get());
            export_gate_bootstrapping_ciphertext_toStream(ofs, value.get(),
                                                          key->params);
            if (--nbits == 0) break;
        }
    }

    assert(nbits <= 0 && "Too small input file");
}

void doDec(const std::string& key_filepath, const std::string& input_filepath,
           const std::string& output_filepath, const std::string& nbits_str)
{
    // nbits may be negative, which means 'no limit about the number of bits'
    auto nbits = std::stoll(nbits_str);

    auto key = import_secret_key(key_filepath);

    std::ifstream ifs{input_filepath, std::ios_base::binary};
    assert(ifs && "Invalid filepath, maybe not exists?");
    std::ofstream ofs{output_filepath, std::ios_base::binary};
    assert(ofs && "Invalid filepath, maybe you don't have right permission?");

    auto value = make_lwe_sample(key);
    while (nbits != 0) {
        if (ifs.peek() == EOF) break;

        unsigned int byte = 0;
        for (int i = 0; i < 8; i++) {
            import_gate_bootstrapping_ciphertext_fromStream(ifs, value.get(),
                                                            key->params);
            byte |= (bootsSymDecrypt(value.get(), key.get()) & 1) << i;
            if (--nbits == 0) break;
        }
        ofs.put(byte);
    }

    assert(nbits <= 0 && "Too small input file");
}

int main(int argc, char** argv)
{
    /*
       tfheutil genkey KEY-FILE
       tfheutil cloudkey INPUT-KEY-FILE OUTPUT-FILE
       tfheutil enc KEY-FILE INPUT-FILE OUTPUT-FILE NUM-OF-BITS
       tfheutil dec KEY-FILE INPUT-FILE OUTPUT-FILE NUM-OF-BITS
    */

    assert(argc >= 3 && "Invalid command-line arguments");

    std::string subcommand = argv[1];
    if (subcommand == "genkey") {
        assert(argc == 3 && "Invalid command-line arguments");
        doGenkey(argv[2]);
    }
    else if (subcommand == "cloudkey") {
        assert(argc == 4 && "Invalid command-line arguments");
        doCloudkey(argv[2], argv[3]);
    }
    else if (subcommand == "enc") {
        assert(argc == 6 && "Invalid command-line arguments");
        doEnc(argv[2], argv[3], argv[4], argv[5]);
    }
    else if (subcommand == "dec") {
        assert(argc == 6 && "Invalid command-line arguments");
        doDec(argv[2], argv[3], argv[4], argv[5]);
    }
    else {
        assert("Invalid command-line arguments");
    }

    return 0;
}
