# Dieses Portfolio

Dieses Portfolio entstand im 5. Semester im Zuge des Moduls "Ausgewählte Themen der Webprogrammierung" als 

## Prämisse

Du bist der große Held. Du hast den bösen Dämonenlord besiegt und das Land feiert dich als ihren Retter.<br/><br/>
Doch jetzt folgt deine wahrscheinlich größte Herausforderung: Die lange Reise nach Hause. Du hast dein Heim schon so lange nicht mehr gesehen. Haben deine Liebsten die Schikanen des Krieges überstanden? Steht dein Heimatdorf überhaupt noch? Dies gilt es herauszufinden.<br/> 
Wenn auch die Dämonen und ihr Lord keine Gefahr mehr sind, so musst du dich nun mit einem neuen Problem herumschlagen: Ruhm. Jeder will etwas von dir und damit verlängert sich die Reise unbestimmt oft auf unbestimmte Zeit, immerhin bist du der Held und kannst nicht einfach nein sagen.<br/>
Wirst du es nach Hause schaffen oder wirst du unterwegs an deinem Alter zu Grunde gehen? Dies gilt es in diesem Reverse Adventure herauszufinden...

## Das Team

**Art:** Elisabeth Seemann und Nando Berg<br/>
**Programming:** Daniel Adam und Nando Berg


## Technische Details

**Engine:** Unity<br/>
**Programmiersprache:** C#<br/>
**Versionsverwaltung:** Github<br/>
**Zustand:** Protyp abgeschlossen, weitere Entwicklung eingestellt<br/>

## Mein Beitrag

Meine Aufgaben in diesem Projekt umfassten die Erstellung der Hintergrundlogik für Dinge wie Charakterbewegung, Alterung und das Questing System.<br/>

### Charakterbewegung

Bei der Charakterbewegung handelt es sich um einen simplen 2D Controller mit Logik für Bewegung nach links und rechts, sowie einem Sprung und einem Angriff.

### Alterung

Die Alterung erfolgt als simple Steigerung eines Integer-Wertes, mit dessen Hilfe mathematisch Dinge wie die Sprunghöhe, Bewegungsgeschwindigkeit und Schaden langsam aber stetig runter skaliert werden. Somit erhält der Spieler ein Gefühl dafür schwächer zu werden.

### Questing System

Das Questing System ist direkt an ein klassisches "Schwarzes Brett" gebunden, an dem Auftrage angenommen werden können. Diese umfassen meist, aber Dinge die nicht nur den Reichtum des Spielers, sondern auch das Alter erhöhen. Dies sorgt dafür, dass Spieler immer die Ressource "Alter" im Hinterkopf behalten müssen und nicht einfach drauf los questen können.