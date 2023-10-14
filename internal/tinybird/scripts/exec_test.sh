
#!/usr/bin/env bash

fail=0;

for t in `find ./tests -name "*.test"`; do
  echo "** Running $t **"
  echo "** $(cat $t)"
  if res=$(bash $t $1 | diff -B ${t}.result -); then
    echo 'OK';
  else
    echo "failed, diff:";
    echo "$res";
    fail=1
  fi
  echo ""
done;

if [ $fail == 1 ]; then
  exit -1;
fi
