#!/bin/bash -xe

case "$1" in
    bump-submodule )
        [ $# -eq 2 ] || ( echo "Usage: $0 bump-submodule SUBMODULE-DIR"; exit 1 )
        git reset
        diff=$(git diff "$2" | grep Subproject | cut -f3 -d' ' | (read l1; read l2; echo -e "$l1\t$l2"))
        echo "$diff" | egrep "^[a-f0-9]{40}	[a-f0-9]{40}$" > /dev/null || ( echo "Invalid diff"; exit 1 )
        git add "$2" && git commit -m \
            "$(echo "$diff" | awk "{print \"Bump $2\",\"from\",\$1,\"to\",\$2}")"
        ;;

    build )
        rm -rf build
        docker build -t kvsp-build .
        docker run -it -v $PWD:/build -w /build kvsp-build:latest
        ;;

    test )
        cd Iyokan && \
            ruby test.rb --skip-preface ../build/bin fast && \
            ruby test.rb --skip-preface ../build/bin cufhe-cahp-ruby-09 && \
            rm _test*
        ;;

    tag )
        [ $# -eq 2 ] || ( echo "Usage: $0 tag VERSION"; exit 1 )
        git tag -a "v$2" -m "v$2"
        ;;

    rebuild-kvsp )
        rm -rf build/kvsp
        docker build -t kvsp-build .
        docker run -it -v $PWD:/build -w /build kvsp-build:latest make kvsp
        build/kvsp/kvsp version
        ;;

    copy )
        [ $# -eq 2 ] || ( echo "Usage: $0 copy VERSION"; exit 1 )
        # Helper: populate a release directory with binaries and licenses.
        # Usage: _copy_destdir DESTDIR IYOKAN_BIN IYOKAN_PACKET_BIN
        _copy_release_dir() {
            local destdir="$1" iyokan_bin="$2" iyokan_packet_bin="$3"
            rm -rf "$destdir"
            mkdir "$destdir"
            cp -a build/bin build/share "$destdir"
            # Place the variant-specific iyokan binaries as the canonical names.
            cp -a "$iyokan_bin"          "$destdir/bin/iyokan"
            cp -a "$iyokan_packet_bin"   "$destdir/bin/iyokan-packet"
            # Remove the variant-suffixed binaries from the release directory.
            rm -f "$destdir"/bin/iyokan-avx2 "$destdir"/bin/iyokan-avx512
            rm -f "$destdir"/bin/iyokan-packet-avx2 "$destdir"/bin/iyokan-packet-avx512
            strip "$destdir"/bin/* || true
            find \
                Alexandrite \
                Iyokan \
                alexandrite-rt \
                cahp-pearl \
                cahp-rt \
                cahp-ruby \
                cahp-sim \
                examples \
                kvsp \
                llvm-cahp \
                yosys \
                -type f -regextype posix-egrep -regex ".*(LICENSE.*|COPYING.*)" | while read line; do \
                    mkdir -p "$destdir/LICENSEs/$(dirname "$line")" &&
                    cp -a "$line" "$destdir/LICENSEs/$(dirname "$line")"/;
                done
            cp -a LICENSE "$destdir/LICENSEs/"
            for file in \
                bin/kvsp \
                bin/iyokan \
                bin/iyokan-packet \
                share/kvsp/alexandrite.toml \
                share/kvsp/alexandrite-core.json \
                share/kvsp/alexandrite-rt/alexandrite.lds \
                share/kvsp/alexandrite-rt/crt0.o \
                share/kvsp/alexandrite-rt/libc.a \
                share/kvsp/cahp-rt/cahp.lds \
                share/kvsp/cahp-rt/crt0.o \
                share/kvsp/cahp-rt/libc.a \
                share/kvsp/pearl-core.json \
                share/kvsp/ruby-core.json \
                ; do
                [ -f "$destdir/$file" ] || ( echo "Missing release artifact: $destdir/$file"; exit 1 )
            done
            "$destdir"/bin/kvsp version
        }
        # AVX512 release (default)
        _copy_release_dir "kvsp_v$2" \
            build/bin/iyokan \
            build/bin/iyokan-packet
        # AVX2 release
        _copy_release_dir "kvsp_v$2-avx2" \
            build/bin/iyokan-avx2 \
            build/bin/iyokan-packet-avx2
        ;;

    pack )
        [ $# -eq 2 ] || ( echo "Usage: $0 pack VERSION"; exit 1 )
        rm -f kvsp.tar.gz kvsp-avx2.tar.gz
        tar -I pigz -cf kvsp.tar.gz     "kvsp_v$2"
        tar -I pigz -cf kvsp-avx2.tar.gz "kvsp_v$2-avx2"
        ;;

    release )
        [ $# -ge 2 ] && [ $# -le 3 ] || ( echo "Usage: $0 release VERSION [NOTES_FILE]"; exit 1 )
        git push --tags
        notes_args=(--generate-notes)
        if [ $# -eq 3 ]; then
            notes_args=(--notes-file "$3")
        fi
        gh release create "v$2" \
            kvsp.tar.gz \
            kvsp-avx2.tar.gz \
            --title "v$2" \
            "${notes_args[@]}"
        ;;

    * )
        echo "Usage: $0 bump-submodule|build|test|tag|rebuild-kvsp|copy|pack|release"
esac
