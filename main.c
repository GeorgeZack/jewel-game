#include <stdio.h>
#include "raylib.h"

// Amethyst and Topaz icons by Skoll under CC BY 3.0
// Emerald, Cut Diamond, Crystalize, and Sapphire icons by Lorc under CC BY 3.0
// Rupee icon by Delapouite under CC BY 3.0

struct gemType {
	Color clr;
	float rot, scl;
	Texture tex;
};

int main(void) {
	const int screenWidth = 800;
	const int screenHeight = 450;
	InitWindow(screenWidth, screenHeight, "test");
	SetTargetFPS(60);

	const int texWidth = 64;
	struct gemType amethyst =	{ .clr = VIOLET,	.rot = 0.0f,	.scl = 1.0f,	.tex = LoadTexture("res/amethyst.png") };
	struct gemType crystal =	{ .clr = YELLOW,	.rot = 0.0f,	.scl = 1.0f,	.tex = LoadTexture("res/crystalize.png") };
	struct gemType diamond =	{ .clr = WHITE,		.rot = 0.0f,	.scl = 1.0f,	.tex = LoadTexture("res/cut-diamond.png") };
	struct gemType emerald =	{ .clr = GREEN,		.rot = 0.0f,	.scl = 1.0f,	.tex = LoadTexture("res/emerald.png") };
	struct gemType rupee =		{ .clr = RED,		.rot = 0.0f,	.scl = 1.0f,	.tex = LoadTexture("res/rupee.png") };
	struct gemType sapphire =	{ .clr = BLUE,		.rot = 0.0f,	.scl = 1.0f,	.tex = LoadTexture("res/saphir.png") };
	struct gemType topaz =		{ .clr = ORANGE,	.rot = 0.0f,	.scl = 1.0f,	.tex = LoadTexture("res/topaz.png") };
	struct gemType gemTypes[] = { amethyst, crystal, diamond, emerald, rupee, sapphire, topaz };
	int gemTypesLength = *(&gemTypes + 1) - gemTypes;

    while (!WindowShouldClose()) {
        BeginDrawing();
		ClearBackground(LIGHTGRAY);
		DrawText("Congrats! You created your first window!", 190, 200, 20, DARKGRAY);
		for(int i = 0; i < gemTypesLength; ++i) {
			struct gemType obj = gemTypes[i];
			DrawTextureEx(obj.tex, (Vector2){ 0 + (i * (texWidth * obj.scl)), 0 }, obj.rot, obj.scl, obj.clr);
		}
        EndDrawing();
    }

	for(int i = 0; i < gemTypesLength; ++i) {
		UnloadTexture(gemTypes[i].tex);
	}
    CloseWindow();
    return 0;
}
