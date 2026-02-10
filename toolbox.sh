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
        git tag -s "v$2" -m "v$2"
        ;;

    rebuild-kvsp )
        rm -rf build/kvsp
        docker build -t kvsp-build .
        docker run -it -v $PWD:/build -w /build kvsp-build:latest make step2_kvsp
        build/kvsp/kvsp version
        ;;

    copy )
        [ $# -eq 2 ] || ( echo "Usage: $0 copy VERSION"; exit 1 )
        destdir="kvsp_v$2"
        mkdir "$destdir"
        cp -a build/bin build/share "$destdir"
        strip "$destdir"/bin/* || true
        find \
            Iyokan \
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
        "$destdir"/bin/kvsp version
        ;;

    pack )
        [ $# -eq 2 ] || ( echo "Usage: $0 pack VERSION"; exit 1 )
        destdir="kvsp_v$2"
        rm -f kvsp.tar.gz
        tar -I pigz -cf kvsp.tar.gz "$destdir"
        ;;

    release )
        [ $# -eq 2 ] || ( echo "Usage: $0 release VERSION"; exit 1 )
        git push --tags
        EDITOR=nano hub release create -a kvsp.tar.gz "v$2"
        ;;

    * )
        echo "Usage: $0 bump-submodule|build|tag|rebuild-kvsp|copy|pack"
esac
