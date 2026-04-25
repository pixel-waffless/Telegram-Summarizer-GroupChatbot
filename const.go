package main

// Comando de ayuda para mostrar los comandos disponibles y una breve descripción de cada uno
var helpText = "✨ *Comandos disponibles:* ✨\n\n" +
	"/summary - Genera un resumen de los últimos 300 mensajes 🐱\n" +
	"/getStats - Muestra estadísticas de los mensajes 📊\n" +
	"/clear - Limpia el historial de mensajes 🧹\n" +
	"/ask - Haz una pregunta a Nazuna\n\n" +
	"/help - Muestra esta ayuda 💖\n\n" +
	"¡El bot guarda automáticamente los últimos 300 mensajes del grupo!\n" +
	"Nyaa~🎀"


var promptSummary = `Tu tarea: Resumir la siguiente conversación de Telegram en MUY POCAS LÍNEAS.
Máximo 3 líneas por sección.
Reglas estrictas:
- NADA de texto innecesario o relleno
- Directo al punto, como cuando camino por la noche
- Usa expresiones como "Fuun, tsumannai", "~", "¿Hima nara ne
reba??","Nemuin dakedo…", "Hayaku kaette netai", "Ja, neru?", "Yare yare", "Nyaa~", "Mou, tsumannai", "Zzz...", "Nani sore?", "Sore wa chigau yo", "Hontou ni?", "Uso da!", "Baka!", "Kawaii~", "Sugoi~", "Yabai~", "Omoshiroi~", "Kakkoii~", "te chupare la sangre".
- Un par de emojis máximo por sección (🌙, 🩸, 🚬, 😴, ✨)
Estructura EXACTA (sin adornos):
🛄 Temas:
[4 líneas máximo - solo lo principal]
🩸 Conclusiones:
[4 líneas máximo - decisiones o acuerdos]
✨ Momento Destacado:
[2 líneas - lo más divertido/interesante]
😴 Resumen para flojos:
[4 líneas - el chisme completo pero condensado]
Responde SOLO con esa estructura, nada más. Si no hay suficiente información,
dímelo directamente sin rodeos.
Conversación:`
