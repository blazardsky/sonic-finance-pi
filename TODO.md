# Roadmap

## v1.2
- [x] mettere un filtro "recenti / anno" anche in entrate (come in spese)
- [x] vista "modal" e tabella semplificata (senza colonna motivo, dettagli e ricevuto da) anche per le entrate (come in spese)
- [x] ordine clienti per entrate
- [x] color coding per le categorie e visualizzazione raggruppata (per ambito di applicazione)

## v1.2.1
Everything shipped since v1.2 (tag `v1.2-debug`), bundled into the next real release.
- [x] Cambio sistema reminder per "fatture questo mese" in home/dashboard: da clienti con fatture negli ultimi 3 mesi a clienti con un contratto in corso e non completamente pagato
- [x] Allineare l'estetica dei pulsanti delle card e dei selettori date
- [x] sistemare modale per la cancellazione delle sotto-categorie (testo e pulsanti escondo dal modal)
- [x] Aggiungere possibilità di modificare i contratti già creati
- [x] Togliere la scritta rossa "da fatturare questo mese" per i contratti dove l'ammontare è > al pattuito nel contratto
- [x] aggiungere l'ammontare nella preview della dashboard "clienti questo mese" per ogni fattura, accanto al nome del cliente
- [x] Aggregare le spese nel tracker per categoria prima che per singolo prodotto
- [x] mettere il tooltip che appare nelle chart come z-index superiore alla legenda
- [x] card su report annuale da rendere più visibili (stile dashboard)
- [x] Fix combobox (es. selezione cliente) non cliccabile su mobile
- [x] Tenere traccia delle quantità di quote possedute per investimento (opzionale, manuale) e del relativo valore/guadagno-perdita nella pagina Titoli

## v1.2.2
- [ ] aggiungere skeletons per caricamento chart (in particolare alla homepage)
- [ ] Aggiungere il tema Alta visibilità / Alto contrasto
- [ ] card su report annuale da spostare su anno
- [ ] Aggiungere nelle ricorrenti anche le entrate, per esempio per uno stipendio fisso
- [ ] pagina successiva / pagina precedente ripetuti anche in fondo alla lista

## v1.3
- [ ] usare tabelle reali (datatable tanstack table) con filtri per colonna

## v1.3.1
- [ ] aggiungere delle chart per le sotto-categorie in report mensile e annuale
- [ ] approfondire la reportistica (e l'esportazione: sankey, viste riassuntive, ...)

## v1.4
- [ ] Aggiungere un mcp per interagire con l'app direttamente da claude code o qualche altra AI

## v1.5
- [ ] Aggiungere un sistema per caricare le spese a partire dalla foto di uno scontrino tramite OCR e probabilmente una AI

## v1.6
- [ ] Aggiungere un sistema per programmare delle spese, capire i momenti migliori per fare acquisti (in base a spese fatte, spese ricorrenti, entrate, stato dei contratti, andamento degli investimenti ecc)

## v1.7
- [ ] drag & drop di widget nella home con possibilità di salvare più profili di visualizzazione
