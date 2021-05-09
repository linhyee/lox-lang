<?php
function fib($n) {
  if ($n < 2) {
    return $n;
  }
  return fib($n-2) + fib($n-1);
}

$start = time();
echo fib(40);
echo "\n";
echo time()-$start;
echo "\n";

