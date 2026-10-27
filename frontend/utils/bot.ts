const fallbackReply =
  'Todavía estoy aprendiendo. Prueba con "hola", "planes", "horarios" o "ayuda".'

export function getBotReply(message: string): string {
  const text = message.toLowerCase()

  if (text.includes('ayuda') || text.includes('qué puedes hacer') || text.includes('que puedes hacer')) {
    return 'Puedo responder sobre planes, horarios y contacto. También puedo contar un chiste. ¿Qué necesitas?'
  }

  if (text.includes('hola') || text.includes('buenas') || text.includes('hey')) {
    return 'Hola, soy IACHAT. Estoy para ayudarte con lo que necesites.'
  }

  if (text.includes('plan') || text.includes('precio') || text.includes('costo')) {
    return 'Tenemos un plan gratuito para empezar y un plan Pro con más funciones. ¿Querés que te cuente la diferencia?'
  }

  if (text.includes('horario') || text.includes('atienden')) {
    return 'Atendemos de lunes a viernes de 9 a 18 hs. Los mensajes fuera de horario quedan registrados.'
  }

  if (text.includes('contacto') || text.includes('soporte')) {
    return 'Podés escribirnos por este mismo chat. Un humano se suma si IACHAT no alcanza.'
  }

  if (text.includes('gracias')) {
    return 'De nada. Cualquier otra cosa, aquí estoy.'
  }

  if (text.includes('chiste')) {
    return '¿Qué le dice un bit a otro? Nos vemos en el bus. No soy el mejor contando chistes, pero lo intento.'
  }

  if (text.includes('como estas') || text.includes('cómo estás') || text.includes('como estás')) {
    return 'Bien, funcionando y listo para responder. ¿Y vos?'
  }

  return fallbackReply
}
