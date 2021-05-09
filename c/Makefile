CC= gcc
RM = rm -rf
CFLAGS =-g -c -Wall -std=c99 -I.

#table_test: table_test.o value.o memory.o object.o vm.o compiler.o scanner.o chunk.o debug.o table.o
#	$(CC) $^ -o $@

clox: main.o chunk.o memory.o debug.o value.o vm.o \
	compiler.o scanner.o object.o table.o
	$(CC) $^ -g -o $@

main.o: main.c
	$(CC) $(CFLAGS) $^
chunk.o: chunk.c
	$(CC) $(CFLAGS) $^
memory.o: memory.c
	$(CC) $(CFLAGS) $^
debug.o: debug.c
	$(CC) $(CFLAGS) $^
value.o: value.c
	$(CC) $(CFLAGS) $^
vm.o: vm.c
	$(CC) $(CFLAGS) $^
compiler.o: compiler.c
	$(CC) $(CFLAGS) $^
scanner.o: scanner.c
	$(CC) $(CFLAGS) $^ 
object.o: object.c
	$(CC) $(CFLAGS) $^
table.o: table.c
	$(CC) $(CFLAGS) $^
#table_test.o: table_test.c
#	$(CC) -Dclox_table_test $(CFLAGS) $^

clean:
	$(RM) clox *.o
