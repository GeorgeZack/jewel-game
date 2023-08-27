#include <stdio.h>
#include "raylib.h"

// Amethyst icon by Skoll under CC BY 3.0
// Emerald, Cut Diamond, and Saphire icons by Lorc under CC BY 3.0
// Rupee icon by Delapouite under CC BY 3.0

int main(void)
{
	const int screenWidth = 800;
	const int screenHeight = 450;
	InitWindow(screenWidth, screenHeight, "test");
	SetTargetFPS(60);

	Texture2D amethyst = LoadTexture("res/amethyst.png");
	Texture2D diamond = LoadTexture("res/cut-diamond.png");
	Texture2D emerald = LoadTexture("res/emerald.png");
	Texture2D rupee = LoadTexture("res/rupee.png");
	Texture2D saphire = LoadTexture("res/saphir.png");
	Texture2D textures[] = { amethyst, diamond, emerald, rupee, saphire };
	int texturesLength = *(&textures + 1) - textures;

    while (!WindowShouldClose())
    {
        BeginDrawing();
		ClearBackground(RAYWHITE);
		DrawText("Congrats! You created your first window!", 190, 200, 20, LIGHTGRAY);
		for(int i = 0; i < texturesLength; ++i) {
			DrawTextureEx(textures[i], (Vector2){ 0 + (i * 200), 0 }, 0.0f, 0.25f, BLUE);
			char temp[10];
			snprintf(temp, sizeof temp, "%d, %d", i, i * 200);
			DrawText(temp, 0, 200 + i * 10, 10, LIGHTGRAY);
		}
        EndDrawing();
    }

	for(int i = 0; i < texturesLength; ++i) {
		UnloadTexture(textures[i]);
	}
    CloseWindow();
    return 0;
}
