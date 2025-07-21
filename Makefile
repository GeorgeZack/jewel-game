CC = gcc
CFLAGS = -Wall -Wextra -O2 -I./raylib-5.5_linux_amd64/include -g -fPIC
LDFLAGS = -L./raylib-5.5_linux_amd64/lib -Wl,-rpath=./raylib-5.5_linux_amd64/lib -lraylib -lGL -lm -lpthread -ldl -lrt -lX11


all: game game_logic.so

game: main.c
	$(CC) $(CFLAGS) -o jewel-game main.c -ldl $(LDFLAGS)

game_logic.so: ./game/game_logic.c
	$(CC) $(CFLAGS) -o $@.tmp ./game/game_logic.c $(LDFLAGS) -shared
	mv $@.tmp $@

clean:
	rm -f jewel-game game_logic.so
